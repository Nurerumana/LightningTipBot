package internal

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
)

func init() {
	stdoutLogger := log.New()
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	stdoutLogger.SetFormatter(customFormatter)

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyTime: "@timestamp",
			log.FieldKeyMsg:  "message",
		},
	})

	log.SetOutput(io.MultiWriter(stdoutLogger.Out, &lumberjack.Logger{
		Filename:   "out.log",
		MaxSize:    1, // megabytes
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true}))

}
