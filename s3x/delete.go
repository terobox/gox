package s3x

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Delete 通过 fullKey 删除文件（公开/私有通用）
func (c *Client) Delete(ctx context.Context, fullKey string) error {
	bucket, s3Key := c.resolve(fullKey)
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	})
	return err
}
