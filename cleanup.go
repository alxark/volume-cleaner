package main

import (
	"github.com/sirupsen/logrus"
	"time"
	"io/ioutil"
	"fmt"
)

type CleanupService struct {
	Directory  string
	Expiration int
	Period     int
	log        *logrus.Logger
}

func NewCleanupService(cleanupDirectory string, expiration int, period int, logger *logrus.Logger) (srv *CleanupService) {
	srv = &CleanupService{
		Directory:  cleanupDirectory,
		Expiration: expiration,
		Period:     period,
		log:        logger,
	}

	logger.Debugf("New logger initialized. Dir: %s", cleanupDirectory)

	return
}

func (s *CleanupService) Run() {
	s.log.Info("Starting cleanup service")

	scanner := NewScanner(s.log)

	for {
		files, err := ioutil.ReadDir(s.Directory)

		if err != nil {
			s.log.Debugf("failed to top-level directory %s, got: %s", s.Directory, err.Error())
			continue
		}

		for _, dir := range files {
			if dir.Name() == "removed" || !dir.IsDir() {
				continue
			}

			s.log.Infof("Scanning %s/%s for subdirectories", s.Directory, dir.Name())

			subFiles, err := ioutil.ReadDir(s.Directory + "/" + dir.Name())
			if err != nil {
				s.log.Debugf("failed to scan for subdirectories in %s/%s", s.Directory, dir.Name())
				continue
			}

			for _, subDir := range subFiles {
				fullName := fmt.Sprintf("%s/%s/%s", s.Directory, dir.Name(), subDir.Name())
				s.log.Debugf("Checking directory %s", fullName)

				expired := scanner.IsExpired(fullName, s.Expiration)
				if !expired {
					s.log.Infof("Directory %s is not expired", fullName)
					continue
				}

				s.log.Infof("Going to move %s to removed folder", fullName)
			}
		}

		s.log.Infof("Scanning finished. Sleep for %d", s.Period)
		time.Sleep(time.Duration(s.Period) * time.Second)
	}

}
