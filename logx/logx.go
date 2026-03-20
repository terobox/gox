package logx

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 封装了 zap.Logger
type Logger struct {
	internal *zap.Logger
	sugar    *zap.SugaredLogger
	cfg      *Config
}

// New 创建新的 Logger 实例
func New(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	cfg.applyDefaults()

	level := ParseLevel(cfg.Level)
	var cores []zapcore.Core

	// === 文件输出（JSON 格式）===
	if !cfg.DisableFile {
		if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
			return nil, fmt.Errorf("logx: 创建日志目录 [%s] 失败: %w", cfg.Dir, err)
		}

		logPath := filepath.Join(cfg.Dir, cfg.Filename)

		writer := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			Compress:   cfg.Compress,
			LocalTime:  true,
		}

		fileEncCfg := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     shanghaiTimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(fileEncCfg),
			zapcore.AddSync(writer),
			level,
		)
		cores = append(cores, fileCore)
	}

	// === 终端输出（人类可读 + 颜色）===
	if cfg.ConsoleOutput {
		consoleEncCfg := zapcore.EncoderConfig{
			TimeKey:          "time",
			LevelKey:         "level",
			NameKey:          "logger",
			CallerKey:        "caller",
			FunctionKey:      zapcore.OmitKey,
			MessageKey:       "msg",
			StacktraceKey:    "stacktrace",
			LineEnding:       zapcore.DefaultLineEnding,
			EncodeLevel:      coloredLevelEncoder,
			EncodeTime:       shanghaiTimeEncoder,
			EncodeDuration:   zapcore.StringDurationEncoder,
			EncodeCaller:     zapcore.ShortCallerEncoder,
			ConsoleSeparator: " | ",
		}

		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncCfg),
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("logx: 至少需要启用一个输出（文件或终端）")
	}

	zapLogger := zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),
		zap.AddCallerSkip(1), // 跳过本层封装，caller 指向实际调用处
	)

	return &Logger{
		internal: zapLogger,
		sugar:    zapLogger.Sugar(),
		cfg:      cfg,
	}, nil
}

// ==================== 生命周期 ====================

// Sync 刷新日志缓冲，应在程序退出前调用
func (l *Logger) Sync() {
	_ = l.internal.Sync()
}

// ==================== 获取底层实例 ====================

// GetZap 获取底层 *zap.Logger
func (l *Logger) GetZap() *zap.Logger {
	return l.internal
}

// GetSugar 获取底层 *zap.SugaredLogger
func (l *Logger) GetSugar() *zap.SugaredLogger {
	return l.sugar
}

// ==================== Structured Logging (zap.Field) ====================

func (l *Logger) Debug(msg string, fields ...zap.Field)  { l.internal.Debug(msg, fields...) }
func (l *Logger) Info(msg string, fields ...zap.Field)   { l.internal.Info(msg, fields...) }
func (l *Logger) Warn(msg string, fields ...zap.Field)   { l.internal.Warn(msg, fields...) }
func (l *Logger) Error(msg string, fields ...zap.Field)  { l.internal.Error(msg, fields...) }
func (l *Logger) DPanic(msg string, fields ...zap.Field) { l.internal.DPanic(msg, fields...) }
func (l *Logger) Panic(msg string, fields ...zap.Field)  { l.internal.Panic(msg, fields...) }
func (l *Logger) Fatal(msg string, fields ...zap.Field)  { l.internal.Fatal(msg, fields...) }

// ==================== Sugar Logging (format) ====================

func (l *Logger) Debugf(template string, args ...interface{})  { l.sugar.Debugf(template, args...) }
func (l *Logger) Infof(template string, args ...interface{})   { l.sugar.Infof(template, args...) }
func (l *Logger) Warnf(template string, args ...interface{})   { l.sugar.Warnf(template, args...) }
func (l *Logger) Errorf(template string, args ...interface{})  { l.sugar.Errorf(template, args...) }
func (l *Logger) DPanicf(template string, args ...interface{}) { l.sugar.DPanicf(template, args...) }
func (l *Logger) Panicf(template string, args ...interface{})  { l.sugar.Panicf(template, args...) }
func (l *Logger) Fatalf(template string, args ...interface{})  { l.sugar.Fatalf(template, args...) }

// ==================== Sugar Logging (key-value) ====================

func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

// ==================== 子 Logger ====================

// With 创建一个携带额外字段的子 Logger
func (l *Logger) With(fields ...zap.Field) *Logger {
	newInternal := l.internal.With(fields...)
	return &Logger{
		internal: newInternal,
		sugar:    newInternal.Sugar(),
		cfg:      l.cfg,
	}
}

// WithValues 使用 key-value 对创建子 Logger
func (l *Logger) WithValues(keysAndValues ...interface{}) *Logger {
	newSugar := l.sugar.With(keysAndValues...)
	return &Logger{
		internal: newSugar.Desugar(),
		sugar:    newSugar,
		cfg:      l.cfg,
	}
}
