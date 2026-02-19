package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	cacheadapter "github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/cache"
	eventadapter "github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/grpc"
	httpadapter "github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/security"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

type Runtime struct {
	cfg        Config
	logger     *slog.Logger
	httpServer *http.Server
	grpcServer *grpc.Server
	grpcLis    net.Listener
	outbox     *eventadapter.OutboxWorker
	cleanupFn  func(context.Context)
}

func NewRuntime(ctx context.Context, configPath string) (*Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("bootstrapping m01 authentication service", "http_port", cfg.HTTPPort, "grpc_port", cfg.GRPCPort)

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.MaxDBConns)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	sqlDB, err := pool.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sql db: %w", err)
	}

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	redisClient, err := cacheadapter.Connect(ctx, cfg.RedisURL)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	repos := postgres.NewRepositories(pool)
	tokenSigner, err := security.NewJWTSigner(cfg.JWTKeyID, cfg.JWTPrivateKeyPEM, cfg.JWTPublicKeyPEM)
	if err != nil {
		if !cfg.AllowEphemeralJWT {
			_ = sqlDB.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("init jwt signer: %w", err)
		}
		logger.Warn("using ephemeral JWT keys for local/dev runtime")
		tokenSigner, err = security.NewEphemeralJWTSigner(cfg.JWTKeyID)
		if err != nil {
			_ = sqlDB.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("init ephemeral jwt signer: %w", err)
		}
	}

	lockouts := cacheadapter.NewRedisLockoutStore(redisClient)
	revocations := cacheadapter.NewRedisSessionRevocationStore(redisClient)
	challenges := cacheadapter.NewRedisMFAChallengeStore(redisClient)
	oidcState := cacheadapter.NewRedisOIDCStateStore(redisClient)
	oidcVerifier := security.NewOIDCVerifier(security.OIDCVerifierConfig{
		HTTPClient: &http.Client{
			Timeout: cfg.OIDCHTTPTimeout,
		},
		Providers: map[string]security.OIDCProviderConfig{
			"google": {
				IssuerURL:    cfg.OIDCGoogleIssuerURL,
				ClientID:     cfg.OIDCGoogleClientID,
				ClientSecret: cfg.OIDCGoogleClientSecret,
				Scopes:       cfg.OIDCGoogleScopes,
			},
		},
	})

	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			DefaultRole:          "INFLUENCER",
			TokenTTL:             cfg.TokenTTL,
			SessionTTL:           cfg.SessionTTL,
			SessionAbsoluteTTL:   cfg.SessionAbsoluteTTL,
			FailedLoginThreshold: cfg.FailedThreshold,
			LockoutDuration:      cfg.LockoutDuration,
		},
		Users:         repos.Users,
		Sessions:      repos.Sessions,
		LoginAttempts: repos.LoginAttempts,
		Outbox:        repos.Outbox,
		Idempotency:   repos.Idempotency,
		Recovery:      repos.Recovery,
		Credentials:   repos.Credentials,
		MFA:           repos.MFA,
		OIDC:          repos.OIDC,
		Lockouts:      lockouts,
		Revocations:   revocations,
		Challenges:    challenges,
		OIDCState:     oidcState,
		OIDCVerifier:  oidcVerifier,
		Hasher:        security.NewBcryptHasher(cfg.BcryptCost),
		TokenSigner:   tokenSigner,
	})

	handler := httpadapter.NewHandler(svc)
	router := httpadapter.NewRouter(handler)
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpc.NewServer()
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	grpcadapter.Register(grpcServer, grpcadapter.NewAuthInternalServer(svc))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		_ = sqlDB.Close()
		_ = redisClient.Close()
		return nil, fmt.Errorf("listen gRPC: %w", err)
	}

	outbox := eventadapter.NewOutboxWorker(
		logger,
		repos.Outbox,
		eventadapter.NewLoggingPublisher(logger),
		cfg.OutboxPollInterval,
		cfg.OutboxBatchSize,
	)

	return &Runtime{
		cfg:        cfg,
		logger:     logger,
		httpServer: httpServer,
		grpcServer: grpcServer,
		grpcLis:    lis,
		outbox:     outbox,
		cleanupFn: func(ctx context.Context) {
			_ = redisClient.Close()
			_ = sqlDB.Close()
		},
	}, nil
}

func (r *Runtime) RunAPI(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		r.logger.Info("http server started", "addr", r.httpServer.Addr)
		if err := r.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()
	go func() {
		r.logger.Info("grpc server started", "addr", r.grpcLis.Addr().String())
		if err := r.grpcServer.Serve(r.grpcLis); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		r.logger.Info("shutdown signal received")
	case err := <-errCh:
		r.logger.Error("server failure", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = r.httpServer.Shutdown(shutdownCtx)
	r.grpcServer.GracefulStop()
	r.cleanupFn(shutdownCtx)
	return nil
}

func (r *Runtime) RunWorker(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	r.logger.Info("outbox worker started")
	err := r.outbox.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.cleanupFn(shutdownCtx)
	return nil
}
