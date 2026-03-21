package mailx

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Config 邮件服务配置
type Config struct {
	Host        string        // SMTP 服务器地址
	Port        int           // SMTP 端口
	Username    string        // 认证用户名
	Password    string        // 认证密码
	FromName    string        // 发件人名称（可选）
	FromAddr    string        // 发件人地址（默认同 Username）
	SSL         bool          // 是否使用直连 SSL（465）
	StartTLS    bool          // 是否使用 STARTTLS（587）
	PoolSize    int           // 连接池大小，默认 5
	Workers     int           // 并发 worker 数，默认 3
	RetryCount  int           // 发送失败重试次数，默认 2
	RetryDelay  time.Duration // 重试间隔，默认 1s
	ConnTimeout time.Duration // 连接超时，默认 10s
	RateLimit   time.Duration // 每封邮件最小间隔，防封禁，默认 100ms
	Logger      Logger        // 外部注入日志器
}

func (c *Config) defaults() {
	if c.PoolSize <= 0 {
		c.PoolSize = 5
	}
	if c.Workers <= 0 {
		c.Workers = 3
	}
	if c.RetryCount < 0 {
		c.RetryCount = 2
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = time.Second
	}
	if c.ConnTimeout <= 0 {
		c.ConnTimeout = 10 * time.Second
	}
	if c.RateLimit <= 0 {
		c.RateLimit = 100 * time.Millisecond
	}
	if c.FromAddr == "" {
		c.FromAddr = c.Username
	}
	if c.Logger == nil {
		c.Logger = &defaultLogger{}
	}
}

// Mailer 邮件发送器
type Mailer struct {
	cfg  *Config
	pool *connPool
	from string
	log  Logger
}

// New 创建 Mailer 实例（在 main 中初始化一次，全局复用）
func New(cfg Config) (*Mailer, error) {
	cfg.defaults()

	from := cfg.FromAddr
	if cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromAddr)
	}

	m := &Mailer{
		cfg:  &cfg,
		pool: newConnPool(&cfg),
		from: from,
		log:  cfg.Logger,
	}

	// 验证连接可用
	sc, err := m.pool.get()
	if err != nil {
		return nil, fmt.Errorf("mailx: initial connection failed: %w", err)
	}
	m.pool.put(sc)
	m.log.Infof("mailx initialized | host=%s:%d pool=%d workers=%d", cfg.Host, cfg.Port, cfg.PoolSize, cfg.Workers)

	return m, nil
}

// Close 关闭连接池
func (m *Mailer) Close() {
	m.pool.close()
	m.log.Infof("mailx closed")
}

// Send 核心发送方法
//
// 传入一个或多个 Group：
//   - 单 Group + 单收件人 = 发一封
//   - 单 Group + 多收件人 = 同一内容独立发送给多人
//   - 多 Group = 不同内容分别发送
//
// 每个收件人独立发送，互相不可见。
func (m *Mailer) Send(groups ...Group) *SendResult {
	start := time.Now()

	// 1. 拆解为独立 task
	var tasks []task
	for gi, g := range groups {
		for _, to := range g.To {
			to = strings.TrimSpace(to)
			if to == "" {
				continue
			}
			tasks = append(tasks, task{
				groupIdx: gi,
				to:       to,
				subject:  g.Subject,
				text:     g.Text,
				html:     g.HTML,
			})
		}
	}

	total := len(tasks)
	if total == 0 {
		return &SendResult{Duration: time.Since(start)}
	}

	// 2. 生产者：投递任务到 channel
	taskCh := make(chan task, total)
	go func() {
		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)
	}()

	// 3. 结果收集
	var (
		successCount int64
		failedCount  int64
		failMu       sync.Mutex
		failures     []FailDetail
	)

	// 4. 定时进度报告
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s := atomic.LoadInt64(&successCount)
				f := atomic.LoadInt64(&failedCount)
				m.log.Infof("progress: %d/%d sent | %d failed | elapsed %s",
					s+f, total, f, time.Since(start).Round(time.Millisecond))
			case <-done:
				return
			}
		}
	}()

	// 5. 消费者：worker 并发处理
	workers := m.cfg.Workers
	if workers > total {
		workers = total
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				err := m.sendOne(t)
				if err != nil {
					atomic.AddInt64(&failedCount, 1)
					failMu.Lock()
					failures = append(failures, FailDetail{To: t.to, Reason: err.Error()})
					failMu.Unlock()
					m.log.Warnf("✗ %s: %s", t.to, err)
				} else {
					atomic.AddInt64(&successCount, 1)
					m.log.Debugf("✓ %s", t.to)
				}
				// 失败/成功都限速
				if m.cfg.RateLimit > 0 {
					time.Sleep(m.cfg.RateLimit)
				}
			}
		}()
	}

	wg.Wait()
	close(done)

	result := &SendResult{
		Total:    total,
		Success:  int(successCount),
		Failed:   int(failedCount),
		Failures: failures,
		Duration: time.Since(start),
	}

	m.log.Infof("completed: %s", result)
	return result
}

func (m *Mailer) discardConn(sc *smtpConn) {
	if sc == nil {
		return
	}
	if err := sc.client.Close(); err != nil {
		m.log.Debugf("close broken smtp conn failed: %v", err)
	}
}

// sendOne 发送单封邮件，带重试
func (m *Mailer) sendOne(t task) error {
	msg := buildMessage(m.from, t.to, t.subject, t.text, t.html)

	var lastErr error
	for attempt := 0; attempt <= m.cfg.RetryCount; attempt++ {
		if attempt > 0 {
			m.log.Debugf("retry %d/%d for %s", attempt, m.cfg.RetryCount, t.to)
			time.Sleep(m.cfg.RetryDelay)
		}

		sc, err := m.pool.get()
		if err != nil {
			lastErr = fmt.Errorf("get conn: %w", err)
			continue
		}

		err = m.doSend(sc, t.to, msg)
		if err != nil {
			lastErr = err
			m.discardConn(sc)
			continue
		}

		// 成功，归还连接
		m.pool.put(sc)
		return nil
	}

	return lastErr
}

// doSend 使用已有连接执行一次发送
func (m *Mailer) doSend(sc *smtpConn, to string, msg []byte) error {
	// 先重置，确保干净状态
	if err := sc.client.Reset(); err != nil {
		return fmt.Errorf("RSET: %w", err)
	}

	if err := sc.client.Mail(m.cfg.FromAddr); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := sc.client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := sc.client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return nil
}
