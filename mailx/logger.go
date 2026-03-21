package mailx

import (
	"fmt"
	"os"
	"time"
)

// Logger 日志接口，外部实现此接口注入即可替换
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// FuncLogger 函数式适配器，方便将全局函数风格的 logger 快速接入
//
// 用法:
//
//	mailx.Config{
//	    Logger: &mailx.FuncLogger{
//	        DebugFunc: logger.Debugf,
//	        InfoFunc:  logger.Infof,
//	        WarnFunc:  logger.Warnf,
//	        ErrorFunc: logger.Errorf,
//	    },
//	}
type FuncLogger struct {
	DebugFunc func(string, ...interface{})
	InfoFunc  func(string, ...interface{})
	WarnFunc  func(string, ...interface{})
	ErrorFunc func(string, ...interface{})
}

func (f *FuncLogger) Debugf(format string, args ...interface{}) {
	if f.DebugFunc != nil {
		f.DebugFunc(format, args...)
	}
}

func (f *FuncLogger) Infof(format string, args ...interface{}) {
	if f.InfoFunc != nil {
		f.InfoFunc(format, args...)
	}
}

func (f *FuncLogger) Warnf(format string, args ...interface{}) {
	if f.WarnFunc != nil {
		f.WarnFunc(format, args...)
	}
}

func (f *FuncLogger) Errorf(format string, args ...interface{}) {
	if f.ErrorFunc != nil {
		f.ErrorFunc(format, args...)
	}
}

// defaultLogger 内置轻量实现，无外部依赖
type defaultLogger struct{}

func (d *defaultLogger) Debugf(format string, args ...interface{}) {}
func (d *defaultLogger) Infof(format string, args ...interface{})  { stdlog("INFO", format, args...) }
func (d *defaultLogger) Warnf(format string, args ...interface{})  { stdlog("WARN", format, args...) }
func (d *defaultLogger) Errorf(format string, args ...interface{}) { stdlog("ERR ", format, args...) }

func stdlog(level, format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, time.Now().Format("15:04:05")+" ["+level+"] "+format+"\n", args...)
}
