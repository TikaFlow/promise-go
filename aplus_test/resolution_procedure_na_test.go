package aplus_test

import (
	"testing"
)

// 2.3.3: Promise 解析流程（对任意 thenable 的完整采纳）。
// 本库刻意只支持 thenable === promise（仅采纳自身 *Promise，见 2.3.2）；
// 不读取任意对象的 .then，无 getter/访问器语义。全部 N/A。
func TestAplus2_3_3(t *testing.T) {
	skipNA(t, "2.3.3.1 retrieving x.then (getter counted once)", 6,
		"Go 无属性访问器/getter，且本库不读取任意对象 .then")
	skipNA(t, "2.3.3.2 retrieving x.then throws", 22,
		"需 getter 抛异常，Go 无对应机制")
	skipNA(t, "2.3.3.3 then is a function, call it", 572,
		"需以 this=x 调用任意 thenable 的 then，本库仅采纳自身 *Promise（2.3.2），不支持 duck-typed thenable")
	skipNA(t, "2.3.3.4 then is not a function, fulfill with x", 10,
		"需'具有非函数 .then 的对象'；等价'以非 thenable 值 fulfill'行为已由 2.3.4 覆盖")
}
