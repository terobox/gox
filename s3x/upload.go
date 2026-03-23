package s3x

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// upload 内部统一上传逻辑（自动检测版）
func (c *Client) upload(ctx context.Context, bucket string, reader io.Reader, opts ...UploadOption) (string, error) {
	o := buildOptions(opts)

	var body io.Reader = reader
	contentType := o.ContentType

	var contentLength *int64
	// 自动检测 Content-Type
	if contentType == "" {
		result, err := detectContentType(reader, o.Filename)
		if err != nil {
			return "", err
		}
		contentType = result.ContentType
		body = result.Body
		if result.ContentLength >= 0 {
			contentLength = &result.ContentLength
		}
	}

	fullKey := c.buildKey(bucket, o.Folder, o.Filename, contentType)
	_, s3Key := c.resolve(fullKey)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(s3Key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if contentLength != nil {
		input.ContentLength = contentLength
	}

	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("upload to %s: %w", bucket, err)
	}

	return fullKey, nil
}

// uploadRaw 原始上传，所有参数由调用方控制
func (c *Client) uploadRaw(ctx context.Context, bucket, folder, filename string, reader io.Reader, contentType string) (string, error) {
	fullKey := c.buildKey(bucket, folder, filename, contentType)
	_, s3Key := c.resolve(fullKey)

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(s3Key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("upload to %s: %w", bucket, err)
	}

	return fullKey, nil
}

// ==================== 公开桶 ====================

// UploadPublic 上传到公开桶，Content-Type 自动检测，Key 自动生成
func (c *Client) UploadPublic(ctx context.Context, reader io.Reader, opts ...UploadOption) (string, error) {
	return c.upload(ctx, c.publicBucket, reader, opts...)
}

// UploadPublicRaw 上传到公开桶，所有参数由调用方完全控制
func (c *Client) UploadPublicRaw(ctx context.Context, folder, filename string, reader io.Reader, contentType string) (string, error) {
	return c.uploadRaw(ctx, c.publicBucket, folder, filename, reader, contentType)
}

// ==================== 私有桶 ====================

// UploadPrivate 上传到私有桶，Content-Type 自动检测，Key 自动生成
func (c *Client) UploadPrivate(ctx context.Context, reader io.Reader, opts ...UploadOption) (string, error) {
	return c.upload(ctx, c.privateBucket, reader, opts...)
}

// UploadPrivateRaw 上传到私有桶，所有参数由调用方完全控制
func (c *Client) UploadPrivateRaw(ctx context.Context, folder, filename string, reader io.Reader, contentType string) (string, error) {
	return c.uploadRaw(ctx, c.privateBucket, folder, filename, reader, contentType)
}
