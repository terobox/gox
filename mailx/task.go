package mailx

import (
	"bytes"
	"fmt"
	"mime"
	"net/textproto"
	"strings"
	"time"

	"math/rand"
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

// buildMessage 构建 MIME 邮件，兼容 text + html (multipart/alternative)
func buildMessage(from, to, subject, text, html string) []byte {
	var buf bytes.Buffer

	boundary := fmt.Sprintf("=_gomailer_%x", rand.Int63())

	// Headers
	headers := textproto.MIMEHeader{}
	headers.Set("From", from)
	headers.Set("To", to)
	headers.Set("Subject", mime.QEncoding.Encode("UTF-8", subject))
	headers.Set("Date", time.Now().Format(time.RFC1123Z))
	headers.Set("MIME-Version", "1.0")
	headers.Set("Message-ID", fmt.Sprintf("<%d.%x@gomailer>", time.Now().UnixNano(), rand.Int63()))

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
		buf.WriteString(text)
		buf.WriteString("\r\n")

		// html part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(html)
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "--\r\n")
	} else if hasHTML {
		headers.Set("Content-Type", "text/html; charset=\"UTF-8\"")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&buf, headers)
		buf.WriteString("\r\n")
		buf.WriteString(html)
	} else {
		headers.Set("Content-Type", "text/plain; charset=\"UTF-8\"")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&buf, headers)
		buf.WriteString("\r\n")
		buf.WriteString(text)
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
