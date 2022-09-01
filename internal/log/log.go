package log

import (
	log "github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"time"
)

func init() {
	log.SetLevel(log.DebugLevel)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	rotateFileHook, err := NewRotateFileHook(RotateFileConfig{
		Filename:   "out.log",
		MaxSize:    1, // megabytes
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
		Level:      log.DebugLevel,
		Formatter:  &ecslogrus.Formatter{},
	})
	log.SetFormatter(customFormatter)
	if err != nil {
		panic(err)
	}
	log.AddHook(rotateFileHook)
}

type Logger struct {
	logger *log.Logger
	vars   map[string]interface{}
}
type Loggable interface {
	Log() map[string]interface{}
}

func WithObjects(objects ...interface{}) *log.Entry {
	fields := log.Fields{}
	for _, object := range objects {
		switch object.(type) {
		case Loggable:
			for key, value := range object.(Loggable).Log() {
				fields[key] = value
			}
		case time.Time:
			t := object.(time.Time)
			fields["runtime"] = time.Now().Sub(t).String()
		case error:
			err := object.(error)
			fields["error"] = err.Error()
		}

	}
	return log.StandardLogger().WithFields(fields)
}

type RotateFileConfig struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	Level      log.Level
	Formatter  log.Formatter
}

type RotateFileHook struct {
	Config    RotateFileConfig
	logWriter io.Writer
}

func NewRotateFileHook(config RotateFileConfig) (log.Hook, error) {

	hook := RotateFileHook{
		Config: config,
	}
	hook.logWriter = &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	return &hook, nil
}

func (hook *RotateFileHook) Levels() []log.Level {
	return log.AllLevels[:hook.Config.Level+1]
}

func (hook *RotateFileHook) Fire(entry *log.Entry) (err error) {
	b, err := hook.Config.Formatter.Format(entry)
	if err != nil {
		return err
	}
	hook.logWriter.Write(b)
	return nil
}
