package s3x

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
)

type detectResult struct {
	ContentType   string
	Body          io.Reader
	ContentLength int64 // -1 表示未知
}

func detectContentType(reader io.Reader, filename string) (*detectResult, error) {
	// 优化路径：如果是 ReadSeeker（如 *os.File），直接嗅探后 Seek 回去
	if rs, ok := reader.(io.ReadSeeker); ok {
		head := make([]byte, 512)
		n, err := rs.Read(head)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read header: %w", err)
		}
		head = head[:n]

		ct := http.DetectContentType(head)
		if ct == "application/octet-stream" && filename != "" {
			if extType := mime.TypeByExtension(filepath.Ext(filename)); extType != "" {
				ct = extType
			}
		}

		// Seek 回起始位置
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek back: %w", err)
		}

		// 获取总长度
		var length int64 = -1
		if end, err := rs.Seek(0, io.SeekEnd); err == nil {
			length = end
			rs.Seek(0, io.SeekStart)
		}

		return &detectResult{
			ContentType:   ct,
			Body:          rs,
			ContentLength: length,
		}, nil
	}

	// 降级路径：纯 io.Reader，读完整内容到内存
	head := make([]byte, 512)
	n, err := io.ReadAtLeast(reader, head, 1)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("read header: %w", err)
	}
	head = head[:n]

	ct := http.DetectContentType(head)
	if ct == "application/octet-stream" && filename != "" {
		if extType := mime.TypeByExtension(filepath.Ext(filename)); extType != "" {
			ct = extType
		}
	}

	// 读取剩余部分，拼成完整 buffer
	rest, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	full := append(head, rest...)

	return &detectResult{
		ContentType:   ct,
		Body:          bytes.NewReader(full),
		ContentLength: int64(len(full)),
	}, nil
}
