package main

import (
	"github.com/sirupsen/logrus"
	"time"
	"io/ioutil"
	"fmt"
	"os"
	"syscall"
	"math"
)

type CleanupService struct {
	Directory  string
	Expiration int
	Period     int
	log        *logrus.Logger
	Scanner    *Scanner
}

func NewCleanupService(cleanupDirectory string, expiration int, period int, logger *logrus.Logger) (srv *CleanupService) {
	srv = &CleanupService{
		Directory:  cleanupDirectory,
		Expiration: expiration,
		Period:     period,
		log:        logger,
	}

	logger.Debugf("New logger initialized. Dir: %s", cleanupDirectory)
	srv.Scanner = NewScanner(srv.log)

	return
}

func (s *CleanupService) Run() {
	s.log.Info("Starting cleanup service")

	removedDir := s.Directory + "/removed"

	if _, err := os.Stat(removedDir); os.IsNotExist(err) {
		os.Mkdir(removedDir, 0766)
		s.log.Infof("Creating removed directories storage at %s", removedDir)
	}

	for {
		files, err := ioutil.ReadDir(s.Directory)

		if err != nil {
			s.log.Debugf("failed to top-level directory %s, got: %s", s.Directory, err.Error())
			continue
		}

		for _, dir := range files {
			if dir.Name() == "removed" {
				continue
			}

			if !dir.IsDir() {
				s.log.Infof("Non directory found on dir level: %s", dir.Name())
				os.Remove(s.Directory + "/" + dir.Name())
				continue
			}

			s.log.Infof("Scanning %s/%s for subdirectories", s.Directory, dir.Name())

			subFiles, err := ioutil.ReadDir(s.Directory + "/" + dir.Name())
			if err != nil {
				s.log.Debugf("failed to scan for subdirectories in %s/%s", s.Directory, dir.Name())
				continue
			}

			for _, subDir := range subFiles {
				if !subDir.IsDir() {
					s.log.Infof("Non directory found on subDir level: %s", subDir.Name())
					os.Remove(s.Directory + "/" + dir.Name())
					continue
				}

				fullName := fmt.Sprintf("%s/%s/%s", s.Directory, dir.Name(), subDir.Name())
				s.log.Debugf("Checking directory %s", fullName)

				expired, bytes := s.Scanner.IsExpired(fullName, s.Expiration)
				if !expired {
					s.log.Infof("Directory %s is not expired", fullName)
					continue
				}

				s.log.Infof("Going to move %s to removed folder, total used bytes: %0.2f MB", fullName, float64(bytes)/(1024*1024))
				newDirName := time.Now().Format("20060102-1504-") + dir.Name() + "_" + subDir.Name()

				curFullPath := s.Directory + "/" + dir.Name() + "/" + subDir.Name()
				newFullPath := removedDir + "/" + newDirName

				s.log.Infof("Moving %s to %s", curFullPath, newFullPath)
				os.Rename(curFullPath, newFullPath)
			}
		}

		s.log.Infof("Scanning finished. Sleep for %d", s.Period)
		time.Sleep(time.Duration(s.Period) * time.Second)
	}

	s.CleanupRemoved()
}

/**
 * Remove dirs which was earlier moved to removed folder
 */
func (s *CleanupService) CleanupRemoved() {
	usage, err := s.GetUsage()
	if err != nil {
		return
	}

	s.log.Infof("Total usage is %0.2f", usage)
	if usage < 0.5 {
		s.log.Infof("Not deleting moved files, total usage below 50 percents")
		return
	}

	coef := int(math.Ceil(10.0 * (1.0 - usage)))
	s.log.Infof("Expiration coefficient is %d", coef)
	removed, err := ioutil.ReadDir(s.Directory + "/removed")

	for _, dir := range removed {
		s.log.Infof("Checking already removed dir: %s", dir.Name())
		fullName := s.Directory + "/removed/" + dir.Name()
		expired, bytes := s.Scanner.IsExpired(fullName, s.Expiration * coef)

		if !expired {
			s.log.Infof("Directory %s is not fully expired", fullName)
			continue
		}

		s.log.Infof("Directory %s is fully expired. Going to free %0.2f MB", fullName, float64(bytes)/(1024*1024))
		os.RemoveAll(fullName)
	}
}

/**
 * Check data directory usage for cleanup
 */
func (s *CleanupService) GetUsage() (float64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(s.Directory, &fs)

	if err != nil {
		return 100, err
	}

	totalBytes := fs.Blocks * uint64(fs.Bsize)
	availBytes := fs.Bfree * uint64(fs.Bsize)
	usingBytes := totalBytes - availBytes

	return float64(usingBytes) / float64(totalBytes) , nil
}