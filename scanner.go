package main

import (
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
)

type Scanner struct {
	log *logrus.Logger
}

func NewScanner(log *logrus.Logger) (scanner *Scanner) {
	scanner = &Scanner{log}

	return scanner
}

// check if there is any new files after expiration
// threshold, if so return yes
func (s *Scanner) IsExpired(directory string, expiration int) bool {
	minimumTime := time.Now().Unix() - int64(expiration)

	files, err := ioutil.ReadDir(directory)
	if err != nil {
		s.log.Debugf("failed to scan directory %s, got: %s", directory, err.Error())
		return true
	}

	for _, file := range files {
		if file.IsDir() {
			if !s.IsExpired(directory+"/"+file.Name(), expiration) {
				return false
			}

			s.log.Debugf("Finished scanning %s/%s for files. No new files found", directory, file.Name())
			continue
		}

		if file.ModTime().Unix() > minimumTime {
			s.log.Infof("file %s/%s has modification time %s", directory, file.Name(), file.ModTime().Format("Jan 2 15:04:05 -0700 MST 2006"))
			return false
		}
	}

	return true
}
