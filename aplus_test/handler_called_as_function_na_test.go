package aplus_test

import (
	"testing"
)

// 2.2.5: 处理器必须以普通函数调用（无 this 绑定）。
// Go 无 this / 严格模式概念，全部 N/A。
func TestAplus2_2_5(t *testing.T) {
	t.Parallel()
	skipNA(t, "2.2.5 handler called as function (this binding)", 4,
		"Go 无 this 绑定与 strict/sloppy 模式概念")
}
