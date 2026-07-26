package idx

import (
	"strings"

	"github.com/google/uuid"
)

// UUID 生成无横线 UUID
func UUID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// UUIDWithPrefix 生成带前缀的无横线 UUID
func UUIDWithPrefix(prefix string) string {
	id := UUID()
	if prefix == "" {
		return id
	}
	return prefix + "_" + id
}

// 示例：
// 7daf3804ff224739b925b8a61b5cc550
// ord_7daf3804ff224739b925b8a61b5cc550
