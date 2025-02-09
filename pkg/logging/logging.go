// pkg/logging/logging.go
package logging

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// SetupLogging initializes the logger with the appropriate settings.
func SetupLogging() *logrus.Logger {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(os.Stdout)
		logger.SetReportCaller(true)

		// Determine log level from environment variable
		logLevel := getLogLevel(os.Getenv("LOG_LEVEL"))
		logger.SetLevel(logLevel)

		// Set formatter based on environment
		if strings.ToLower(os.Getenv("LOG_FORMAT")) == "json" {
			logger.SetFormatter(&logrus.JSONFormatter{
				DisableTimestamp: false,
				PrettyPrint:      false,
			})
		} else {
			customFormatter := &logrus.TextFormatter{
				DisableTimestamp: true, // Let k8s handle timestamps
				FullTimestamp:    false,
			}
			logger.SetFormatter(customFormatter)
		}

		logger.Debug("Logger initialized with level: ", logLevel.String())
	}

	return logger
}

// getLogLevel returns the logrus log level based on the input string.
func getLogLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logrus.DebugLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}
