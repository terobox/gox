# mailx

高性能 SMTP 邮件发送工具库，基于连接池 + 生产者/消费者模型。

## 特性

- **连接池复用**：启动时建立 SMTP 连接，池化管理，避免每封邮件重新握手认证
- **生产者-消费者**：任务拆解后投入 channel，多 worker 并发消费，充分利用 goroutine
- **并发控制**：可配置 worker 数量和发送间隔，防止触发上游限流/封禁
- **自动重试**：连接失败或发送失败自动重试，次数和间隔可配
- **内容兼容**：支持纯文本、HTML、或两者同时（multipart/alternative，客户端自动选择最佳格式）
- **收件人隔离**：每个收件人独立发送，互相不可见
- **实时进度**：发送过程中定时输出进度，结束返回汇总结果
- **日志可替换**：实现 `Logger` 接口即可注入你自己的日志器

## 配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Host | string | - | SMTP 服务器地址 |
| Port | int | - | 端口（465/587/25） |
| Username | string | - | 认证用户名 |
| Password | string | - | 认证密码 |
| FromName | string | "" | 发件人显示名 |
| FromAddr | string | =Username | 发件人地址 |
| SSL | bool | false | 直连 SSL（端口 465） |
| StartTLS | bool | false | STARTTLS（端口 587） |
| PoolSize | int | 5 | 连接池大小 |
| Workers | int | 3 | 并发 worker 数 |
| RetryCount | int | 2 | 失败重试次数 |
| RetryDelay | Duration | 1s | 重试间隔 |
| ConnTimeout | Duration | 10s | 连接超时 |
| RateLimit | Duration | 100ms | 每封邮件最小发送间隔 |
| Logger | Logger | 内置 | 自定义日志器 |

## 核心逻辑

```
Send(groups) 
  → 拆解为独立 task (每个收件人一封)
  → task 投入 channel (生产者)
  → N 个 worker goroutine 消费 (消费者)
    → 从连接池获取连接
    → 构建 MIME 邮件 (multipart/alternative)
    → 执行 SMTP 发送
    → 失败则重试，连接损坏则丢弃重建
    → 成功则归还连接到池
  → 定时报告进度
  → 全部完成返回 SendResult
```

## 调用方式

```go
// 初始化一次
m, _ := mailx.New(mailx.Config{...})
defer m.Close()

// 一个人，一份内容
m.Send(mailx.Group{To: []string{"a@x.com"}, Subject: "hi", Text: "hello"})

// 多个人，同一内容（独立发送）
m.Send(mailx.Group{To: []string{"a@x.com","b@x.com"}, Subject: "hi", HTML: "<b>hello</b>"})

// 多组，不同内容
m.Send(
    mailx.Group{To: []string{"a@x.com"}, Subject: "A的通知", Text: "for A"},
    mailx.Group{To: []string{"b@x.com","c@x.com"}, Subject: "BC的通知", HTML: "<p>for BC</p>"},
)
```

## 自定义日志器

```go
type MyLogger struct{}
func (l *MyLogger) Debugf(f string, a ...interface{}) { ... }
func (l *MyLogger) Infof(f string, a ...interface{})  { ... }
func (l *MyLogger) Warnf(f string, a ...interface{})  { ... }
func (l *MyLogger) Errorf(f string, a ...interface{}) { ... }

m, _ := mailx.New(mailx.Config{
    Logger: &MyLogger{},
    ...
})
```

## 终端输出示例

```
14:30:01 [INFO] mailx initialized | host=smtp.example.com:465 pool=5 workers=3
14:30:03 [INFO] progress: 15/50 sent | 1 failed | elapsed 2.001s
14:30:05 [INFO] progress: 38/50 sent | 1 failed | elapsed 4.003s
14:30:06 [WARN] ✗ bad@invalid.com: RCPT TO: 550 user not found
14:30:06 [INFO] completed: Total: 50 | Success: 49 | Failed: 1 | Duration: 5.12s
                Failures:
                  - bad@invalid.com: RCPT TO: 550 user not found
```
```

---

## 架构总结

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Send()     │     │   channel    │     │  Worker 1    │──→ Pool.get() → SMTP Send → Pool.put()
│  拆解 Group │────→│   taskCh     │────→│  Worker 2    │──→ Pool.get() → SMTP Send → Pool.put()
│  为 task    │     │              │     │  Worker 3    │──→ Pool.get() → SMTP Send → Pool.put()
└─────────────┘     └──────────────┘     └──────────────┘
     生产者              缓冲管道              消费者
                                                │
                                    ┌───────────┘
                                    ▼
                          失败 → 重试(N次) → 记录 FailDetail
                          成功 → atomic 计数
                                    │
                          定时 ticker → 输出进度
                                    │
                          全部完成 → 返回 SendResult
```