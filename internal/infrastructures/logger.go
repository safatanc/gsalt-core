package infrastructures

import (
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	return logger
}
