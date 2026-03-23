package s3x

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// PresignURL 生成预签名下载链接（通常用于私有桶）
func (c *Client) PresignURL(ctx context.Context, fullKey string, expiry time.Duration) (string, error) {
	bucket, s3Key := c.resolve(fullKey)
	presigned, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}
	return presigned.URL, nil
}
