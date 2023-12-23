package main

import (
	"flag"
	"github.com/alxark/volume-cleaner/internal"
	"log/slog"
	"os"
	"sync"
)

var (
	cleanupDirectory string
	expiration       int
	period           int
	metricsAddr      string
)

func init() {
	flag.StringVar(&cleanupDirectory, "cleanupDirectory", "/data/", "configuration file")
	flag.IntVar(&expiration, "expiration", 86400, "number of seconds to wait before marking directory as expired")
	flag.IntVar(&period, "period", 3600, "period between cleanup checks")
	flag.StringVar(&metricsAddr, "metricsAddr", "127.0.0.1:23456", "metrics port")
	flag.Parse()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("application initialized", "expirationTime", expiration)

	metrics, err := internal.NewMetrics(metricsAddr, logger)
	if err != nil {
		logger.Error("failed to initialize metrics", "error", err.Error())
		os.Exit(1)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("starting metrics service")
		_ = metrics.Run()
		logger.Info("metrics service stopped")
	}()

	cleanupService := internal.NewCleanupService(cleanupDirectory, expiration, period, logger)
	wg.Add(1)
	go func() {
		defer wg.Done()

		cleanupService.Run()
		logger.Info("cleanup service stopped")
	}()

	wg.Wait()
}
