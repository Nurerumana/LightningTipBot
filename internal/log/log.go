package log

import (
	"github.com/LightningTipBot/LightningTipBot/internal/telegram/intercept"
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
	//log.StandardLogger().ReportCaller = true
	log.SetFormatter(customFormatter)
	if err != nil {
		panic(err)
	}
	log.AddHook(rotateFileHook)
}

// Logger is printing Text format of log map to stdout. Will also write to file using logrus hook
type Logger struct {
	logger *log.Logger
	vars   map[string]interface{}
}

// Loggable structs provider their own logmap.
type Loggable interface {
	Log() map[string]interface{}
}

func WithObjects(objects ...interface{}) *log.Entry {
	fields := log.Fields{}
	for _, object := range objects {
		switch object.(type) {
		case intercept.Context:
			ctx := object.(intercept.Context)
			if fields["path"] != "" {
				switch {
				case ctx.Message() != nil:
					fields["path"] = "message"
				case ctx.Query() != nil:
					fields["path"] = "query"
				case ctx.Callback() != nil:
					fields["path"] = "callback"
				}
			}
			// check for context uuid
			if uuid := ctx.Value("uuid"); uuid != nil {
				fields["uuid"] = uuid.(string)
			}
			// add chat info from context
			if chat := ctx.Chat(); chat != nil {
				fields["chat_id"] = ctx.Chat().ID
				fields["title"] = ctx.Chat().Title
			}
			// add text as data if not set yet
			if fields["data"] == "" {
				fields["data"] = ctx.Text()
			}
		case Loggable:
			// loggable structs provider their own logmap
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

// RotateFileConfig configuration for logfile rotation.
type RotateFileConfig struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	Level      log.Level
	Formatter  log.Formatter
}

// RotateFileHook
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
