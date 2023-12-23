package internal

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"math"
	"os"
	"syscall"
	"time"
)

var volumesSizesMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "vc",
	Subsystem: "cleanup",
	Name:      "volume_size",
	Help:      "volume size in bytes",
}, []string{"volume"})

var usageMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "vc",
	Subsystem: "cleanup",
	Name:      "usage",
	Help:      "volume usage",
}, []string{"storage"})

func init() {
	prometheus.MustRegister(volumesSizesMetric)
}

type CleanupService struct {
	Directory  string
	Expiration int
	Period     int
	log        *slog.Logger
	Scanner    *Scanner
}

func NewCleanupService(cleanupDirectory string, expiration int, period int, logger *slog.Logger) (srv *CleanupService) {
	srv = &CleanupService{
		Directory:  cleanupDirectory,
		Expiration: expiration,
		Period:     period,
		log:        logger,
	}

	logger.Info("new logger initialized", "dir", cleanupDirectory)
	srv.Scanner = NewScanner(srv.log)

	return
}

func (s *CleanupService) Run() {
	volumesHistory := make(map[string]bool)

	s.log.Info("Starting cleanup service")

	removedDir := s.Directory + "/removed"

	if _, err := os.Stat(removedDir); os.IsNotExist(err) {
		os.Mkdir(removedDir, 0766)
		s.log.Info("creating removed directories storage", "dir", removedDir)
	}

	for {
		files, err := os.ReadDir(s.Directory)

		if err != nil {
			s.log.Debug("failed to top-level directory", "dir", s.Directory, "error", err.Error())
			continue
		}

		// flush volume history values to zero
		for k, _ := range volumesHistory {
			volumesHistory[k] = false
		}

		for _, dir := range files {
			if dir.Name() == "removed" {
				continue
			}

			if !dir.IsDir() {
				s.log.Info("non directory found on dir level", "dir", dir.Name())
				os.Remove(s.Directory + "/" + dir.Name())
				continue
			}

			s.log.Info("scanning for subdirectories", "dir", s.Directory, "subDir", dir.Name())

			subFiles, err := os.ReadDir(s.Directory + "/" + dir.Name())
			if err != nil {
				s.log.Debug("failed to scan for subdirectories", "dir", s.Directory, "subDir", dir.Name())
				continue
			}

			if len(subFiles) == 0 {
				os.Remove(s.Directory + "/" + dir.Name())
				s.log.Info("directory is empty, removing it", "dir", dir.Name())
				continue
			}

			for _, subDir := range subFiles {
				if !subDir.IsDir() {
					s.log.Info("non directory found on subDir level", "dir", dir.Name(), "file", subDir.Name())
					os.Remove(s.Directory + "/" + dir.Name() + "/" + subDir.Name())
					continue
				}

				fullName := fmt.Sprintf("%s/%s/%s", s.Directory, dir.Name(), subDir.Name())
				s.log.Debug("checking directory", "dir", fullName)

				expired, bytes, _ := s.Scanner.IsExpired(fullName, s.Expiration)

				volumesHistory[dir.Name()+"/"+subDir.Name()] = true
				volumesSizesMetric.WithLabelValues(dir.Name() + "/" + subDir.Name()).Set(float64(bytes))

				if !expired {
					s.log.Info("directory is not expired", "dir", fullName)
					continue
				}

				s.log.Info("going to move to removed folder", "dir", fullName, "sizeBytes", float64(bytes)/(1024*1024))
				newDirName := time.Now().Format("20060102-1504-") + dir.Name() + "_" + subDir.Name()

				curFullPath := s.Directory + "/" + dir.Name() + "/" + subDir.Name()
				newFullPath := removedDir + "/" + newDirName

				s.log.Info("moving directory", "srcDir", curFullPath, "dstDir", newFullPath)
				os.Rename(curFullPath, newFullPath)
			}
		}

		for k, active := range volumesHistory {
			if active {
				volumesSizesMetric.WithLabelValues(k).Set(0.0)
			}
		}

		s.log.Info("removing already moved data")
		err = s.CleanupRemoved()
		if err != nil {
			s.log.Info("failed to remove old data, got", "error", err.Error())
		}

		s.log.Info("scanning finished", "sleepPeriod", s.Period)
		time.Sleep(time.Duration(s.Period) * time.Second)
	}
}

/**
 * Remove dirs which was earlier moved to removed folder
 */
func (s *CleanupService) CleanupRemoved() error {
	usage, err := s.GetUsage()
	if err != nil {
		return err
	}

	usageMetric.WithLabelValues(s.Directory).Set(usage)

	s.log.Info("total usage calculated", "usage", usage)
	if usage < 0.5 {
		s.log.Info("not deleting moved files, total usage below 50 percents")
		return nil
	}

	coef := int(math.Ceil(10.0 * (1.0 - usage)))
	s.log.Info("expiration coefficient calculated", "coef", coef)
	removed, err := os.ReadDir(s.Directory + "/removed")

	for _, dir := range removed {
		s.log.Info("checking already removed dir", "dir", dir.Name())
		fullName := s.Directory + "/removed/" + dir.Name()
		expired, bytes, _ := s.Scanner.IsExpired(fullName, s.Expiration*coef)

		if !expired {
			s.log.Info("directory is not fully expired", "dir", fullName)
			continue
		}

		s.log.Info("directory is fully expired", "dir", fullName, "sizeBytes", float64(bytes)/(1024*1024))
		err = os.RemoveAll(fullName)
		if err != nil {
			s.log.Info("failed to remove", "dir", fullName, "error", err.Error())
		}
	}

	return nil
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

	return float64(usingBytes) / float64(totalBytes), nil
}
