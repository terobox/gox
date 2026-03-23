package s3x

type UploadOptions struct {
	Folder      string // 子目录，如 "avatar"、"document"
	Filename    string // 原始文件名，仅用于提取扩展名
	ContentType string // 手动指定 Content-Type，留空则自动检测
}

type UploadOption func(*UploadOptions)

// WithFolder 指定存储子目录
func WithFolder(folder string) UploadOption {
	return func(o *UploadOptions) {
		o.Folder = folder
	}
}

// WithFilename 传入原始文件名（用于提取扩展名）
func WithFilename(name string) UploadOption {
	return func(o *UploadOptions) {
		o.Filename = name
	}
}

// WithContentType 手动指定 Content-Type，跳过自动检测
func WithContentType(ct string) UploadOption {
	return func(o *UploadOptions) {
		o.ContentType = ct
	}
}

func buildOptions(opts []UploadOption) UploadOptions {
	var o UploadOptions
	for _, fn := range opts {
		fn(&o)
	}
	return o
}
