# s3x

轻量级 S3 客户端封装，提供公开/私有双桶上传、自动 Content-Type 检测、预签名 URL 生成等常用功能。

## 目录结构

```
s3x
├── client.go    # 客户端初始化与配置
├── upload.go    # 上传（公开桶 / 私有桶 / 自动检测 / Raw 模式）
├── delete.go    # 删除文件
├── presign.go   # 生成预签名下载链接
├── url.go       # 拼接公开访问 URL
├── detect.go    # Content-Type 自动检测
├── key.go       # Key 生成与解析
├── option.go    # 上传选项（Folder / Filename / ContentType）
```

## 功能

- **双桶模型**：内置公开桶 + 私有桶，一套代码覆盖两种场景
- **自动 Content-Type 检测**：基于文件内容嗅探 + 扩展名回退，支持 `io.ReadSeeker` 零拷贝优化
- **Key 自动生成**：`{bucket}/{prefix}/{folder}/{uuid}{ext}`，避免文件名冲突
- **预签名 URL**：为私有桶文件生成带过期时间的临时下载链接
- **公开 URL 拼接**：直接生成可访问的公开链接
- **Raw 模式**：完全由调用方控制所有参数，不做自动检测
- **兼容 S3 协议**：支持 AWS S3、MinIO、Cloudflare R2 等

## 安装

```bash
go get github.com/your-org/gox/s3x
```

## 快速开始

### 初始化客户端

```go
client, err := s3x.New(ctx, &s3x.Config{
    Endpoint:      "https://s3.amazonaws.com",
    AccessKey:     "your-access-key",
    SecretKey:     "your-secret-key",
    Region:        "us-east-1",       // 默认 us-east-1
    PublicBucket:  "my-public",
    PrivateBucket: "my-private",
    PathPrefix:    "app/v1",          // 可选，所有 Key 的统一前缀
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 上传到公开桶

```go
file, _ := os.Open("photo.jpg")
defer file.Close()

fullKey, err := client.UploadPublic(ctx, file,
    s3x.WithFolder("avatar"),
    s3x.WithFilename("photo.jpg"),
)
// fullKey: "my-public/app/v1/avatar/550e8400-e29b-41d4-a716-446655440000.jpg"
```

### 上传到私有桶

```go
fullKey, err := client.UploadPrivate(ctx, file,
    s3x.WithFolder("document"),
    s3x.WithFilename("report.pdf"),
)
```

### 手动指定 Content-Type

```go
fullKey, err := client.UploadPublic(ctx, reader,
    s3x.WithFolder("data"),
    s3x.WithFilename("export.csv"),
    s3x.WithContentType("text/csv; charset=utf-8"),
)
```

### Raw 模式上传（完全控制参数）

```go
fullKey, err := client.UploadPublicRaw(ctx, "avatar", "photo.jpg", reader, "image/jpeg")

fullKey, err := client.UploadPrivateRaw(ctx, "document", "report.pdf", reader, "application/pdf")
```

### 获取公开访问 URL

```go
url := client.PublicURL(fullKey)
// https://s3.amazonaws.com/my-public/app/v1/avatar/550e8400-...jpg
```

### 生成预签名下载链接

```go
url, err := client.PresignURL(ctx, fullKey, 15*time.Minute)
// 带签名的临时下载链接，15 分钟后过期
```

### 删除文件

```go
err := client.Delete(ctx, fullKey)
```

## Key 格式说明

所有上传方法返回的 `fullKey` 格式为：

```
{bucket}/{pathPrefix}/{folder}/{uuid}{ext}
```

`fullKey` 可直接传给 `Delete`、`PresignURL`、`PublicURL`，内部自动解析出 bucket 和 s3Key。