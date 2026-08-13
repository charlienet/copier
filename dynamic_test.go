package copier

// dynamic_test.go：Copy 动态能力回归测试。
// 用途：证明 Copy[S, R] 在类型参数实例化为 any（动态类型）时，等价覆盖
// 传统 Copy(src, dst).Do() 的全部动态类型能力——框架可用 Copy 统一入口，
// 无需保留 Copy 的动态转发场景（实验验证结论，2026-08）。
// 每个场景对照 Copy 直调结果断言等价；如未来发现差异，在此处记录并标记 TODO。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dynUser struct {
	Name string
	Age  int
	Tags []string
}

// ============ 场景 1：map→map 动态值 ============

func TestDynamicMapToMap(t *testing.T) {
	m1 := map[string]any{"a": 1, "b": []int{1, 2}}

	var m2, m3 map[string]any
	assert.NoError(t, Copy(m1, &m2).Do()) // S/R 均为 map[string]any
	assert.NoError(t, Copy(m1, &m3).Do())
	assert.Equal(t, m3, m2)
	assert.Equal(t, m1, m2)
}

// ============ 场景 2：框架转发（src/dst 形参为 any） ============

// 框架转发形态 A：dst 形参为 any，经 &dst（*any）反射填充。
func dynamicForwardAny(src, dst any) error {
	return Copy(src, &dst).Do()
}

// 框架转发形态 B：dst 为具体指针（框架泛型化 dst 类型，src 保持动态 any）。
func dynamicForwardPtr[R any](src any, dst *R) error {
	return Copy(src, dst).Do()
}

func TestDynamicForward(t *testing.T) {
	src := dynUser{Name: "n", Age: 1, Tags: []string{"a"}}

	// 形态 A1：dst 为 any 变量，经 &dst 反射填充（S=dynUser, R=any）
	var dstAny any
	assert.NoError(t, Copy(src, &dstAny).Do())
	got, ok := dstAny.(dynUser)
	assert.True(t, ok, "dstAny 应被填充为 dynUser")
	assert.Equal(t, src, got)

	// 形态 A2：全动态（S=any, R=any）
	var srcAny any = src
	var dstAny2 any
	assert.NoError(t, Copy(srcAny, &dstAny2).Do())
	got2, ok := dstAny2.(dynUser)
	assert.True(t, ok)
	assert.Equal(t, src, got2)

	// 形态 B：框架转发（S=any, R=dynUser）
	var d dynUser
	assert.NoError(t, dynamicForwardPtr(src, &d))
	assert.Equal(t, src, d)
}

// ============ 场景 3：struct→map（动态 src） ============

func TestDynamicStructToMap(t *testing.T) {
	src := dynUser{Name: "n", Age: 1, Tags: []string{"a"}}
	var srcAny any = src

	var m1, m2 map[string]any
	assert.NoError(t, Copy(srcAny, &m1).Do()) // S=any → struct→map 全链
	assert.NoError(t, Copy(src, &m2).Do())
	assert.Equal(t, m2, m1)
	assert.Equal(t, []string{"a"}, m1["Tags"])
}

// ============ 场景 4：动态 + 链式选项 ============

func TestDynamicWithOptions(t *testing.T) {
	src := dynUser{Name: "n", Tags: []string{"a"}} // Age 零值

	// 形态 A：dst 为 any 变量——Interface 分支整体赋值，IgnoreEmpty 对整体赋值不生效；
	// 但两入口行为一致（同形态同内核）
	var d1, d2 any
	assert.NoError(t, Copy(src, &d1).IgnoreEmpty().Do())
	assert.NoError(t, Copy(src, &d2).IgnoreEmpty().Do())
	assert.Equal(t, d2, d1)
	assert.Equal(t, src, d1.(dynUser)) // Age 零值仍在（整体赋值）

	// 形态 B：dst 具体类型——逐字段拷贝，IgnoreEmpty 生效（零值跳过，保留初始值）
	var d3, d4 dynUser
	d3.Age, d4.Age = 99, 99
	assert.NoError(t, Copy(src, &d3).IgnoreEmpty().Do())
	assert.NoError(t, Copy(src, &d4).IgnoreEmpty().Do())
	assert.Equal(t, d4, d3)
	assert.Equal(t, 99, d3.Age)
}

// ============ 场景 5：nil dst 语义 ============

func TestDynamicNilDst(t *testing.T) {
	src := dynUser{Name: "n"}

	// nil 指针 dst：顶层自动分配，两入口一致
	var p1, p2 *dynUser
	assert.NoError(t, Copy(src, &p1).Do())
	assert.NoError(t, Copy(src, &p2).Do())
	assert.NotNil(t, p1)
	assert.Equal(t, p2, p1)

	// any 变量 dst：两入口一致
	var a1, a2 any
	assert.NoError(t, Copy(src, &a1).Do())
	assert.NoError(t, Copy(src, &a2).Do())
	assert.Equal(t, a2, a1)
	assert.Equal(t, src, a1.(dynUser))

	// nil 源（动态）：两入口均 ErrInvalidCopyFrom（nilSrcZero 默认关闭）
	var srcNil *dynUser
	var dst1, dst2 dynUser
	err1 := Copy(srcNil, &dst1).Do() // S=*dynUser
	err2 := Copy(srcNil, &dst2).Do()
	assert.True(t, errors.Is(err1, ErrInvalidCopyFrom))
	assert.True(t, errors.Is(err2, ErrInvalidCopyFrom))
}

// ============ 场景 6：error 语义 ============

func TestDynamicErrorSemantics(t *testing.T) {
	type badSrc struct{ Title string }
	type badDst struct{ Title int }
	src := badSrc{Title: "abc"}

	var srcAny any = src
	var d1, d2 badDst
	err1 := Copy(srcAny, &d1).Do() // 默认 strict
	err2 := Copy(src, &d2).Do()

	assert.True(t, errors.Is(err1, ErrConversionFailed))
	assert.True(t, errors.Is(err2, ErrConversionFailed))
	assert.Equal(t, err2.Error(), err1.Error()) // 错误消息（含字段路径）一致
	assert.Contains(t, err1.Error(), "Title:")
}
