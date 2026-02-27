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

	eventadapter "github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/adapters/grpc"
	httpadapter "github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/application"
	"google.golang.org/grpc"
)

type Runtime struct {
	cfg        Config
	logger     *slog.Logger
	httpServer *http.Server
	grpcServer *grpc.Server
	grpcLis    net.Listener
	worker     *eventadapter.Worker
}

func NewRuntime(ctx context.Context, configPath string) (*Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With("service", cfg.ServiceID)
	slog.SetDefault(logger)

	repos := postgres.NewRepositories()
	dlqPublisher := eventadapter.NewLoggingDLQPublisher()

	service := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:    cfg.ServiceID,
			IdempotencyTTL: cfg.IdempotencyTTL,
			EventDedupTTL:  cfg.EventDedupTTL,
		},
		Warehouse:   repos.Warehouse,
		Exports:     repos.Exports,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Voting:      grpcadapter.NewVotingClient(cfg.VotingGRPCURL),
		Social:      grpcadapter.NewSocialClient(cfg.SocialGRPCURL),
		Tracking:    grpcadapter.NewTrackingClient(cfg.TrackingGRPCURL),
		Submission:  grpcadapter.NewSubmissionClient(cfg.SubmissionGRPCURL),
		Finance:     grpcadapter.NewFinanceClient(cfg.FinanceGRPCURL),
		DLQ:         dlqPublisher,
	})

	handler := httpadapter.NewHandler(service)
	router := httpadapter.NewRouter(handler)
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpc.NewServer()
	grpcadapter.Register(grpcServer, grpcadapter.NewAnalyticsInternalServer(service))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return nil, err
	}

	consumer := eventadapter.NewMemoryConsumer()
	worker := eventadapter.NewWorker(logger, consumer, dlqPublisher, service, cfg.ConsumerPollInterval)

	_ = ctx
	return &Runtime{
		cfg:        cfg,
		logger:     logger,
		httpServer: httpServer,
		grpcServer: grpcServer,
		grpcLis:    lis,
		worker:     worker,
	}, nil
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
	return nil
}

func (r *Runtime) RunWorker(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		if err := r.worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}
