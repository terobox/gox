package s3x

import (
	"strings"

	"github.com/google/uuid"
)

// UUID v4 去横线 (最安全，兼容性最好)
// 输出: "a1b2c3d4e5f67890a1b2c3d4e5f67890" (32字符)
func uuidNoDash() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
