// Command server is the entry point for the BeeBase statistics-service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	appstatistics "github.com/sbezhuk/beebase-statistics-service/internal/application/statistics"
	"github.com/sbezhuk/beebase-statistics-service/internal/config"
	"github.com/sbezhuk/beebase-statistics-service/internal/platform/apiaryclient"
	"github.com/sbezhuk/beebase-statistics-service/internal/platform/harvestclient"
	"github.com/sbezhuk/beebase-statistics-service/internal/platform/hiveclient"
	"github.com/sbezhuk/beebase-statistics-service/internal/platform/inspectionclient"
	transporthttp "github.com/sbezhuk/beebase-statistics-service/internal/transport/http"
	statisticshttp "github.com/sbezhuk/beebase-statistics-service/internal/transport/http/statistics"

	"github.com/sbezhuk/beebase-common/authmw"
	"github.com/sbezhuk/beebase-common/logger"
	"github.com/sbezhuk/beebase-common/server"
	"github.com/sbezhuk/beebase-common/sessionstore"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// .env is optional: present in local dev, absent in production/containers.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisConnectCtx, cancelRedisConnect := context.WithTimeout(ctx, cfg.RedisConnectTimeout)
	redisClient, err := sessionstore.NewRedisClient(redisConnectCtx, cfg.RedisAddr)
	cancelRedisConnect()
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	defer redisClient.Close()

	log.Info("connected to redis")

	sessions := sessionstore.NewStore(redisClient)

	// Fails fast at boot if auth-service's JWKS endpoint isn't reachable,
	// consistent with how every other service handles this same
	// dependency; docker-compose orders auth-service before this service
	// accordingly.
	verifier, err := authmw.NewVerifierFromJWKSURL(ctx, cfg.AuthJWKSURL, sessions)
	if err != nil {
		return fmt.Errorf("build JWKS verifier: %w", err)
	}

	apiaries := apiaryclient.New(cfg.ApiaryServiceURL)
	hives := hiveclient.New(cfg.HiveServiceURL)
	inspections := inspectionclient.New(cfg.InspectionServiceURL)
	harvests := harvestclient.New(cfg.HarvestServiceURL)
	statisticsService := appstatistics.NewService(apiaries, hives, inspections, harvests)
	statisticsHandler := statisticshttp.NewHandler(statisticsService, log)

	router := transporthttp.NewRouter(log, statisticsHandler, verifier)

	srv := server.New(server.Config{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	})

	errCh := make(chan error, 1)
	go func() {
		log.Info("starting http server", "port", cfg.HTTPPort, "env", cfg.Env)
		errCh <- srv.Run()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("run server: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	log.Info("server stopped cleanly")
	return nil
}
