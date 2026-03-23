package s3x

import (
	"mime"
	"path"
	"path/filepath"
	"strings"
)

// buildKey 生成 fullKey: "{bucket}/{prefix}/{folder}/{uuid}{ext}"
func (c *Client) buildKey(bucket, folder, filename, contentType string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		exts, _ := mime.ExtensionsByType(contentType)
		if len(exts) > 0 {
			ext = exts[0]
		}
	}
	s3Key := path.Join(c.pathPrefix, folder, uuidNoDash()+ext)
	return bucket + "/" + s3Key
}

// resolve 从 fullKey 解析出 bucket 和 s3Key
func (c *Client) resolve(fullKey string) (bucket, s3Key string) {
	parts := strings.SplitN(fullKey, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}
