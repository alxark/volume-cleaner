package main

import (
	"github.com/sirupsen/logrus"
	"flag"
)

var (
	cleanupDirectory string
	expiration       int
	period           int
)

func init() {
	flag.StringVar(&cleanupDirectory, "cleanupDirectory", "/data/", "configuration file")
	flag.IntVar(&expiration, "expiration", 86400, "run service in verification mode")
	flag.IntVar(&period, "period", 3600, "period between cleanup checks")
	flag.Parse()
}

func main() {
	logger := logrus.New()

	formatter := &logrus.TextFormatter{FullTimestamp: true}
	logrus.SetFormatter(formatter)
	logger.Level = logrus.DebugLevel

	logger.Infof("Application initialized, expiration: %d", expiration)

	cleanupService := NewCleanupService(cleanupDirectory, expiration, period, logger)
	cleanupService.Run()
}
