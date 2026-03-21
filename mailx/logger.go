package mailx

import (
	"fmt"
	"os"
	"time"
)

// Logger 轻量日志接口，支持外部注入替换
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type defaultLogger struct{}

func (d *defaultLogger) Debugf(format string, args ...interface{}) {}
func (d *defaultLogger) Infof(format string, args ...interface{}) {
	stdlog("INFO", format, args...)
}
func (d *defaultLogger) Warnf(format string, args ...interface{}) {
	stdlog("WARN", format, args...)
}
func (d *defaultLogger) Errorf(format string, args ...interface{}) {
	stdlog("ERR ", format, args...)
}

func stdlog(level, format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, time.Now().Format("15:04:05")+" ["+level+"] "+format+"\n", args...)
}
