package mailx

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"sync"
	"time"
)

// smtpConn 包装一个 SMTP 连接
type smtpConn struct {
	client    *smtp.Client
	createdAt time.Time
}

// connPool SMTP 连接池
type connPool struct {
	mu      sync.Mutex
	conns   chan *smtpConn
	factory func() (*smtpConn, error)
	maxSize int
	maxAge  time.Duration
	closed  bool
}

func newConnPool(cfg *Config) *connPool {
	p := &connPool{
		conns:   make(chan *smtpConn, cfg.PoolSize),
		maxSize: cfg.PoolSize,
		maxAge:  5 * time.Minute,
	}
	p.factory = func() (*smtpConn, error) {
		// addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

		var c *smtp.Client
		var err error

		if cfg.SSL {
			tlsConfig := &tls.Config{ServerName: cfg.Host}
			conn, e := tls.DialWithDialer(&net.Dialer{Timeout: cfg.ConnTimeout}, "tcp", addr, tlsConfig)
			if e != nil {
				return nil, fmt.Errorf("tls dial: %w", e)
			}
			c, err = smtp.NewClient(conn, cfg.Host)
		} else {
			conn, e := net.DialTimeout("tcp", addr, cfg.ConnTimeout)
			if e != nil {
				return nil, fmt.Errorf("tcp dial: %w", e)
			}
			c, err = smtp.NewClient(conn, cfg.Host)
			if err == nil && cfg.StartTLS {
				tlsConfig := &tls.Config{ServerName: cfg.Host}
				if e := c.StartTLS(tlsConfig); e != nil {
					c.Close()
					return nil, fmt.Errorf("starttls: %w", e)
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("smtp client: %w", err)
		}

		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err = c.Auth(auth); err != nil {
			c.Close()
			return nil, fmt.Errorf("auth: %w", err)
		}

		return &smtpConn{client: c, createdAt: time.Now()}, nil
	}
	return p
}

// get 从池中获取连接，没有则新建
func (p *connPool) get() (*smtpConn, error) {
	for {
		select {
		case sc := <-p.conns:
			// 检查连接是否过期
			if time.Since(sc.createdAt) > p.maxAge {
				sc.client.Close()
				continue
			}
			// 检查连接是否存活 (NOOP)
			if err := sc.client.Noop(); err != nil {
				sc.client.Close()
				continue
			}
			return sc, nil
		default:
			return p.factory()
		}
	}
}

// put 归还连接到池中
func (p *connPool) put(sc *smtpConn) {
	if sc == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		sc.client.Close()
		return
	}
	p.mu.Unlock()

	select {
	case p.conns <- sc:
	default:
		// 池满，丢弃
		sc.client.Close()
	}
}

// close 关闭池中所有连接
func (p *connPool) close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	close(p.conns)
	for sc := range p.conns {
		sc.client.Close()
	}
}
