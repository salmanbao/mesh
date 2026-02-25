package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/adapters/cache"
	eventadapter "github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/adapters/grpc"
	httpadapter "github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/adapters/security"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Runtime struct {
	cfg        Config
	logger     *slog.Logger
	httpServer *http.Server
	grpcServer *grpc.Server
	grpcLis    net.Listener
	outbox     *eventadapter.OutboxWorker
	consumer   *eventadapter.ConsumerWorker
	authClient *grpcadapter.AuthClient
	cleanupFn  func(context.Context)
}

func NewRuntime(ctx context.Context, configPath string) (*Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With("service", cfg.ServiceID)
	slog.SetDefault(logger)

	db, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.MaxDBConns)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := postgres.RunMigrations(ctx, db); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	redisClient, err := cache.Connect(ctx, cfg.RedisURL)
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	cacheStore := cache.NewRedisCache(redisClient)

	authClient, err := grpcadapter.NewAuthClient(ctx, cfg.AuthGRPCURL)
	if err != nil {
		_ = sqlDB.Close()
		_ = redisClient.Close()
		return nil, err
	}

	repos := postgres.NewRepositories(db)
	service := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:                         cfg.ServiceID,
			ProfileCacheTTL:                     cfg.ProfileCacheTTL,
			UsernameCooldownDays:                cfg.UsernameCooldownDays,
			UsernameRedirectDays:                cfg.UsernameRedirectDays,
			MaxSocialLinks:                      cfg.MaxSocialLinks,
			IdempotencyTTL:                      cfg.IdempotencyTTL,
			EventDedupTTL:                       cfg.EventDedupTTL,
			FeatureProfileCompletenessVisible:   cfg.FeatureProfileCompletenessVisible,
			FeatureKYCReverificationInterval:    cfg.FeatureKYCReverificationInterval,
			FeatureFollowerSyncHighEarnerHourly: cfg.FeatureFollowerSyncHighEarnerHourly,
			FeaturePayPalOwnershipVerification:  cfg.FeaturePayPalOwnershipVerification,
			FeatureAvatarManualRetry:            cfg.FeatureAvatarManualRetry,
			KYCAnonymizeAfter:                   cfg.KYCAnonymizeAfter,
			UsernameHistoryRetention:            cfg.UsernameHistoryRetention,
		},
		Profiles:          repos.Profiles,
		SocialLinks:       repos.SocialLinks,
		PayoutMethods:     repos.PayoutMethods,
		KYC:               repos.KYC,
		Stats:             repos.Stats,
		ReservedUsernames: repos.ReservedUsernames,
		UsernameHistory:   repos.UsernameHistory,
		Reads:             repos.Reads,
		Outbox:            repos.Outbox,
		EventDedup:        repos.EventDedup,
		Idempotency:       repos.Idempotency,
		AuthClient:        authClient,
		Cache:             cacheStore,
		Encryption:        security.NewAESGCMEncryption(cfg.EncryptionSeed),
	})

	handler := httpadapter.NewHandler(service)
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
	grpcadapter.Register(grpcServer, grpcadapter.NewProfileInternalServer(service))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		_ = redisClient.Close()
		_ = sqlDB.Close()
		_ = authClient.Close()
		return nil, err
	}

	publisher := ports.EventPublisher(eventadapter.NewLoggingPublisher(logger))
	consumerAdapter := eventadapter.Consumer(eventadapter.NewNoopConsumer())
	var closers []io.Closer
	if len(cfg.KafkaBrokers) > 0 {
		kafkaPublisher, pubErr := eventadapter.NewKafkaPublisher(cfg.KafkaBrokers, map[string]string{
			"user.profile_updated": cfg.KafkaTopicProfileUpdated,
		})
		if pubErr != nil {
			logger.WarnContext(ctx, "kafka publisher disabled, using logging publisher", "error", pubErr)
		} else {
			publisher = kafkaPublisher
			closers = append(closers, kafkaPublisher)
		}

		kafkaConsumer, conErr := eventadapter.NewKafkaConsumer(
			cfg.KafkaBrokers,
			cfg.KafkaConsumerGroup,
			[]string{cfg.KafkaTopicUserRegistered, cfg.KafkaTopicUserDeleted},
		)
		if conErr != nil {
			logger.WarnContext(ctx, "kafka consumer disabled, using noop consumer", "error", conErr)
		} else {
			consumerAdapter = kafkaConsumer
			closers = append(closers, kafkaConsumer)
		}
	}
	outbox := eventadapter.NewOutboxWorker(logger, repos.Outbox, publisher, cfg.OutboxPollInterval, cfg.OutboxBatchSize)
	consumer := eventadapter.NewConsumerWorker(logger, consumerAdapter, service, cfg.ConsumerPollInterval)

	return &Runtime{
		cfg:        cfg,
		logger:     logger,
		httpServer: httpServer,
		grpcServer: grpcServer,
		grpcLis:    lis,
		outbox:     outbox,
		consumer:   consumer,
		authClient: authClient,
		cleanupFn: func(ctx context.Context) {
			for _, closer := range closers {
				_ = closer.Close()
			}
			_ = authClient.Close()
			_ = redisClient.Close()
			_ = sqlDB.Close()
		},
	}, nil
}

func Build(ctx context.Context, configPath string) (*Runtime, error) {
	return NewRuntime(ctx, configPath)
}

func (r *Runtime) RunAPI(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 2)

	go func() {
		if err := r.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		if err := r.grpcServer.Serve(r.grpcLis); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		r.logger.ErrorContext(ctx, "runtime failure", "error", err)
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
	errCh := make(chan error, 2)

	go func() {
		if err := r.outbox.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()
	go func() {
		if err := r.consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		r.cleanupFn(context.Background())
		return err
	}
}
