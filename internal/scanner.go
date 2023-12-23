package internal

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Scanner struct {
	log *slog.Logger
}

func NewScanner(log *slog.Logger) (scanner *Scanner) {
	scanner = &Scanner{log}

	return scanner
}

// check if there is any new files after expiration
// threshold, if so return yes
func (s *Scanner) IsExpired(directory string, expiration int) (expired bool, size int64, err error) {
	expired = true

	minimumTime := time.Now().Unix() - int64(expiration)
	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.log.Error("failed accessing file", "path", path, "error", err.Error())
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}

		if info.ModTime().Unix() > minimumTime {
			expired = false
		}

		return nil
	})

	if err != nil {
		s.log.Error("failed to walk directory", "dir", directory, "error", err.Error())
		return false, 0, err
	}

	return expired, size, nil
}
