# logx — 基于 Zap 的日志模块

基于 [go.uber.org/zap](https://github.com/uber-go/zap) 封装的通用日志模块，
搭配 [lumberjack](https://github.com/natefinch/lumberjack) 实现日志轮转。

## 特性

- **双输出**：终端（人类可读 + 彩色 Level）+ 文件（JSON 格式，方便日志分析）
- **日志轮转**：按文件大小切割，支持保留天数、备份数量、gzip 压缩
- **时区固定**：`Asia/Shanghai`
- **零配置可用**：所有参数都有合理默认值，传 `nil` 即可开箱使用
- **Caller 准确**：日志行号指向实际业务调用位置，而非封装层
- **灵活集成**：提供 `GetZap()` 获取原生 `*zap.Logger`，可对接 Gin、gRPC 等框架

## 安装依赖

```bash
go get go.uber.org/zap
go get gopkg.in/natefinch/lumberjack.v2
```

## 快速开始

**1. 最小运行示例**

```go
package main

import (
	"github.com/terobox/gox/logx"
)

func main() {
	logger, err := logx.New(nil)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("hello")
}
```

**2. 在业务项目中全局使用（推荐）**

在业务项目中创建一个薄封装层，注册全局实例：

```go
// infra/logger/logger.go
package logger

import (
	"sync"

	"github.com/terobox/subaway/backend/infra/logx"
	"go.uber.org/zap"
)

// 全局 Logger 实例
var (
	global *logx.Logger
	once   sync.Once
)

// Init 初始化全局 Logger（项目启动时调用一次）
// 传入 nil 则使用全部默认配置
func Init(cfg *logx.Config) {
	once.Do(func() {
		l, err := logx.New(cfg)
		if err != nil {
			panic("logger 初始化失败: " + err.Error())
		}
		global = l
	})
}

// GetLogger 获取全局 Logger 实例
func GetLogger() *logx.Logger {
	ensureInit()
	return global
}

// GetZap 获取底层 zap.Logger（用于 gin、gRPC 等框架集成）
func GetZap() *zap.Logger {
	ensureInit()
	return global.GetZap()
}

// Sync 刷新缓冲，程序退出前调用
func Sync() {
	if global != nil {
		global.Sync()
	}
}

// ensureInit 确保已初始化，未初始化时用默认配置自动初始化
func ensureInit() {
	if global == nil {
		Init(nil)
	}
}

// ==================== 全局快捷方法 ====================

func Debug(msg string, fields ...zap.Field)       { ensureInit(); global.Debug(msg, fields...) }
func Debugf(template string, args ...interface{}) { ensureInit(); global.Debugf(template, args...) }
func Debugw(msg string, kv ...interface{})        { ensureInit(); global.Debugw(msg, kv...) }

func Info(msg string, fields ...zap.Field)       { ensureInit(); global.Info(msg, fields...) }
func Infof(template string, args ...interface{}) { ensureInit(); global.Infof(template, args...) }
func Infow(msg string, kv ...interface{})        { ensureInit(); global.Infow(msg, kv...) }

func Warn(msg string, fields ...zap.Field)       { ensureInit(); global.Warn(msg, fields...) }
func Warnf(template string, args ...interface{}) { ensureInit(); global.Warnf(template, args...) }
func Warnw(msg string, kv ...interface{})        { ensureInit(); global.Warnw(msg, kv...) }

func Error(msg string, fields ...zap.Field)       { ensureInit(); global.Error(msg, fields...) }
func Errorf(template string, args ...interface{}) { ensureInit(); global.Errorf(template, args...) }
func Errorw(msg string, kv ...interface{})        { ensureInit(); global.Errorw(msg, kv...) }

func Fatal(msg string, fields ...zap.Field)       { ensureInit(); global.Fatal(msg, fields...) }
func Fatalf(template string, args ...interface{}) { ensureInit(); global.Fatalf(template, args...) }

func Panic(msg string, fields ...zap.Field)       { ensureInit(); global.Panic(msg, fields...) }
func Panicf(template string, args ...interface{}) { ensureInit(); global.Panicf(template, args...) }

// With 创建带字段的子 Logger
func With(fields ...zap.Field) *logx.Logger {
	ensureInit()
	return global.With(fields...)
}

// WithValues 使用 kv 对创建子 Logger
func WithValues(keysAndValues ...interface{}) *logx.Logger {
	ensureInit()
	return global.WithValues(keysAndValues...)
}
```

接着 `main.go` 中初始化：

```go
// main.go
// 初始化 Logger（一般从配置文件读取，这里示例直接写）
logger.Init(&logx.Config{
	Level:         "debug",
	Dir:           "logs",
	Filename:      "gateway.log",
	MaxSize:       200,
	MaxAge:        14,
	MaxBackups:    5,
	Compress:      true,
	ConsoleOutput: true,
})
defer logger.Sync()

logger.Debugf("logger 初始化完成，接着可以在任意地方调用")
logger.Warnf("配置项 %s 未设置，使用默认值: %d", "timeout", 30)
logger.Info("服务启动成功", zap.String("addr", ":8080"))
```

然后业务代码中任意位置：

```go
// everywhere
import "your-project/infra/logger"

logger.Info("用户登录成功", zap.String("uid", "12345"))
logger.Debugf("耗时: %v", elapsed)
```

## 配置参数说明

| 参数            | 类型   | 默认值      | 说明                                     |
| --------------- | ------ | ----------- | ---------------------------------------- |
| `Level`         | string | `"info"`    | 日志级别：debug/info/warn/error/fatal    |
| `Dir`           | string | `"logs"`    | 日志文件目录（支持相对/绝对路径）        |
| `Filename`      | string | `"app.log"` | 日志文件名                               |
| `MaxSize`       | int    | `100`       | 单文件最大体积（MB），超过后触发轮转     |
| `MaxAge`        | int    | `30`        | 旧日志保留天数                           |
| `MaxBackups`    | int    | `10`        | 最多保留的旧日志文件数量                 |
| `Compress`      | bool   | `true`      | 是否对轮转后的旧日志进行 gzip 压缩       |
| `ConsoleOutput` | bool   | `true`      | 是否同时输出到终端                       |
| `DisableFile`   | bool   | `false`     | 是否禁用文件输出（开发阶段可能只看终端） |

## 输出格式

### 终端（人类可读，Level 带颜色）

```
2026-03-20 16:09:33 +08:00 | DEBUG | middleware/base.go:27 | 请求路径: /v1/chat/completions
2026-03-20 16:09:33 +08:00 | INFO  | service/user.go:42 | 用户创建成功
2026-03-20 16:09:33 +08:00 | ERROR | dao/mysql.go:88 | 数据库连接失败
```

### 文件（JSON 格式，方便 ELK/日志分析）

```json
{"level":"DEBUG","time":"2026-03-20 16:09:33 +08:00","caller":"middleware/base.go:27","msg":"请求路径: /v1/chat/completions"}
{"level":"INFO","time":"2026-03-20 16:09:33 +08:00","caller":"service/user.go:42","msg":"用户创建成功"}
```

## 日志轮转

由 lumberjack 按文件大小切割，轮转后文件命名：

```
logs/
├── gateway.log                                  # 当前日志
├── gateway-2026-03-18T23-59-59.000.log.gz      # 已压缩
├── gateway-2026-03-19T12-30-00.000.log.gz
└── gateway-2026-03-20T08-00-00.000.log.gz
```

## API 一览

### 结构化日志（高性能，推荐高频路径使用）

```go
log.Debug(msg, zap.String("key", "val"), zap.Int("code", 200))
log.Info(msg, fields...)
log.Warn(msg, fields...)
log.Error(msg, fields...)
log.Fatal(msg, fields...)   // 调用后 os.Exit(1)
log.Panic(msg, fields...)   // 调用后 panic
```

### 格式化日志（方便，适合调试）

```go
log.Debugf("用户 %s 请求了 %s", name, path)
log.Infof("耗时: %v", duration)
log.Errorf("失败原因: %v", err)
```

### 键值对日志

```go
log.Infow("GIN 请求",
    "method", "POST",
    "status", 200,
    "path", "/api/v1/users",
)
```

### 子 Logger（携带公共字段）

```go
reqLog := log.With(zap.String("request_id", "abc123"))
reqLog.Info("开始处理")
reqLog.Info("处理完成")
// 两条日志都自动携带 request_id
```

### 框架集成

```go
// 获取原生 zap.Logger，用于 Gin、gRPC 等
zapLogger := log.GetZap()
```

## 最终目录

```
infra/logx/
├── config.go           # 配置结构 + 默认值
├── encoder.go          # shanghaiTimeEncoder + coloredLevelEncoder
├── level.go            # ParseLevel
├── logx.go             # 核心：New() + Logger 方法（原 logger.go）
└── README.md           # 文档
```