package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type processResult struct {
	name string
	err  error
}

func runProcesses(
	processCtx context.Context,
	signalCtx context.Context,
	cancelProcesses context.CancelFunc,
	server *http.Server,
	workerPool interface{ Run(context.Context) error },
	appConfig applicationConfig,
	logger *slog.Logger,
) error {
	results := make(chan processResult, 2)
	runningProcesses := 1
	go serveHTTP(server, appConfig.HTTP.Address, logger, results)

	if appConfig.Worker.Enabled {
		runningProcesses++
		go serveWorkers(processCtx, workerPool, appConfig.Worker.Concurrency, logger, results)
	} else {
		logger.Warn("generation worker pool is disabled")
	}

	var runErr error
	completedProcesses := 0
	select {
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	case result := <-results:
		completedProcesses++
		runErr = unexpectedProcessError(result)
	}
	cancelProcesses()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), appConfig.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("shutdown http server: %w", err))
	}
	for completedProcesses < runningProcesses {
		select {
		case result := <-results:
			completedProcesses++
			if result.err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("%s: %w", result.name, result.err))
			}
		case <-shutdownCtx.Done():
			return errors.Join(runErr, shutdownCtx.Err())
		}
	}

	logger.Info("application shutdown completed")
	return runErr
}

func serveHTTP(server *http.Server, address string, logger *slog.Logger, results chan<- processResult) {
	logger.Info("http server started", "address", address)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	results <- processResult{name: "http server", err: err}
}

func serveWorkers(
	ctx context.Context,
	workerPool interface{ Run(context.Context) error },
	concurrency int,
	logger *slog.Logger,
	results chan<- processResult,
) {
	logger.Info("generation worker pool started", "concurrency", concurrency)
	results <- processResult{name: "generation worker pool", err: workerPool.Run(ctx)}
}

func unexpectedProcessError(result processResult) error {
	if result.err != nil {
		return fmt.Errorf("%s: %w", result.name, result.err)
	}
	return fmt.Errorf("%s stopped unexpectedly", result.name)
}
