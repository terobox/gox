package s3x

import "fmt"

// PublicURL 拼接公开访问 URL
func (c *Client) PublicURL(fullKey string) string {
	return fmt.Sprintf("%s/%s", c.endpoint, fullKey)
}
