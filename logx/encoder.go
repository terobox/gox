package logx

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

// coloredLevelEncoder 终端彩色 Level 编码
func coloredLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color string
	switch level {
	case zapcore.DebugLevel:
		color = colorPurple
	case zapcore.InfoLevel:
		color = colorCyan
	case zapcore.WarnLevel:
		color = colorYellow
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		color = colorRed
	default:
		color = colorReset
	}
	enc.AppendString(fmt.Sprintf("%s%s%s", color, level.CapitalString(), colorReset))
}

// shanghaiTimeEncoder 上海时区时间编码
func shanghaiTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	enc.AppendString(t.In(shanghai).Format("2006-01-02 15:04:05 -07:00"))
}
