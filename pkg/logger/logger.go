package logger

import (
	"log"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func GetLogger() zerolog.Logger {
	logLevel := zerolog.TraceLevel

	if os.Getenv("LOG_LEVEL") != "" {
		level := os.Getenv("LOG_LEVEL")

		switch level {
		case "trace":
			logLevel = zerolog.TraceLevel
		case "debug":
			logLevel = zerolog.DebugLevel
		case "info":
			logLevel = zerolog.InfoLevel
		case "warn":
			logLevel = zerolog.WarnLevel
		case "error":
			logLevel = zerolog.ErrorLevel
		case "fatal":
			logLevel = zerolog.FatalLevel
		case "panic":
			logLevel = zerolog.PanicLevel
		default:
			log.Fatalf("Unknown log level: %s", level)
		}
	}

	logger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(logLevel).With().Timestamp().Caller().Logger()

	return logger
}
