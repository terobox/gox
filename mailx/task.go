package mailx

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"crypto/rand"
)

// Group 一组邮件：同一内容发送给多个收件人（每人独立）
type Group struct {
	To      []string // 收件人列表
	Subject string   // 主题
	Text    string   // 纯文本内容（可选）
	HTML    string   // HTML 内容（可选）
}

// task 内部任务单元：一封邮件
type task struct {
	groupIdx int
	to       string
	subject  string
	text     string
	html     string
}

// FailDetail 单封失败详情
type FailDetail struct {
	To     string
	Reason string
}

// SendResult 发送汇总结果
type SendResult struct {
	Total    int
	Success  int
	Failed   int
	Failures []FailDetail
	Duration time.Duration
}

func (r *SendResult) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Total: %d | Success: %d | Failed: %d | Duration: %s",
		r.Total, r.Success, r.Failed, r.Duration.Round(time.Millisecond)))
	if len(r.Failures) > 0 {
		b.WriteString("\nFailures:")
		for _, f := range r.Failures {
			b.WriteString(fmt.Sprintf("\n  - %s: %s", f.To, f.Reason))
		}
	}
	return b.String()
}

func extractDomain(fromAddr string) string {
	// 兼容 "Name <user@example.com>" 这种格式
	if addr, err := mail.ParseAddress(fromAddr); err == nil && addr.Address != "" {
		fromAddr = addr.Address
	}

	at := strings.LastIndex(fromAddr, "@")
	if at == -1 || at == len(fromAddr)-1 {
		return "localhost"
	}

	domain := strings.TrimSpace(fromAddr[at+1:])
	if domain == "" {
		return "localhost"
	}

	return domain
}

func buildMessageID(fromAddr string) string {
	domain := extractDomain(fromAddr)
	return fmt.Sprintf("<%d.%x@%s>", time.Now().UnixNano(), randomHex(8), domain)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// 极少发生，兜底
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func writeQuotedPrintable(buf *bytes.Buffer, s string) {
	w := quotedprintable.NewWriter(buf)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
}

// buildMessage 构建 MIME 邮件，兼容 text + html (multipart/alternative)
func buildMessage(from, to, subject, text, html string) []byte {
	var buf bytes.Buffer

	boundary := fmt.Sprintf("=_mailx_%x", randomHex(16))

	// Headers
	headers := textproto.MIMEHeader{}
	headers.Set("From", from)
	headers.Set("To", to)
	headers.Set("Subject", mime.QEncoding.Encode("UTF-8", subject))
	headers.Set("Date", time.Now().Format(time.RFC1123Z))
	headers.Set("MIME-Version", "1.0")
	// headers.Set("Message-ID", fmt.Sprintf("<%d.%x@mailx>", time.Now().UnixNano(), rand.Int63()))
	headers.Set("Message-ID", buildMessageID(from))

	hasText := text != ""
	hasHTML := html != ""

	if hasText && hasHTML {
		headers.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=\"%s\"", boundary))
		writeHeaders(&buf, headers)
		buf.WriteString("\r\n")

		// text part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		// buf.WriteString(text)
		writeQuotedPrintable(&buf, text)
		buf.WriteString("\r\n")

		// html part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		// buf.WriteString(html)
		writeQuotedPrintable(&buf, html)
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "--\r\n")
	} else if hasHTML {
		headers.Set("Content-Type", "text/html; charset=\"UTF-8\"")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&buf, headers)
		buf.WriteString("\r\n")
		// buf.WriteString(html)
		writeQuotedPrintable(&buf, html)
	} else {
		headers.Set("Content-Type", "text/plain; charset=\"UTF-8\"")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&buf, headers)
		buf.WriteString("\r\n")
		// buf.WriteString(text)
		writeQuotedPrintable(&buf, text)
	}

	return buf.Bytes()
}

func writeHeaders(buf *bytes.Buffer, h textproto.MIMEHeader) {
	// 固定顺序输出关键头
	order := []string{"From", "To", "Subject", "Date", "Message-Id", "MIME-Version", "Content-Type", "Content-Transfer-Encoding"}
	written := map[string]bool{}
	for _, k := range order {
		if v := h.Get(k); v != "" {
			buf.WriteString(k + ": " + v + "\r\n")
			written[k] = true
		}
	}
	for k, vs := range h {
		if written[textproto.CanonicalMIMEHeaderKey(k)] {
			continue
		}
		for _, v := range vs {
			buf.WriteString(k + ": " + v + "\r\n")
		}
	}
}
