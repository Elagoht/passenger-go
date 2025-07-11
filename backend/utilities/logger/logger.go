package logger

import (
	"log"
	"os"
	"sync"
)

var (
	logger     *log.Logger
	loggerOnce sync.Once
)

func GetLogger() *log.Logger {
	loggerOnce.Do(func() {
		logger = log.New(os.Stdout, "[Passenger] ", log.LstdFlags)
	})
	return logger
}
