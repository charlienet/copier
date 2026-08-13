package copier

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type strictSrc struct {
	Num string
}

type strictDst struct {
	Num int
}

// ============ v0.2 起默认严格模式 ============

func TestStrictDefaultOn(t *testing.T) {
	// 无选项时解析失败默认报错（默认翻转验证）
	src := strictSrc{Num: "abc"}
	var dst strictDst

	err := Copy(src, &dst).Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))

	// Lenient 显式退出严格模式
	dst = strictDst{}
	err = Copy(src, &dst).Lenient().Do()
	assert.NoError(t, err)
	assert.Equal(t, 0, dst.Num)
}

// ============ 宽松模式（Lenient）：解析失败静默留零，不报错 ============

func TestStrictDefaultLenient(t *testing.T) {
	src := strictSrc{Num: "abc"}
	var dst strictDst

	err := Copy(src, &dst).Lenient().Do()
	assert.NoError(t, err)
	assert.Equal(t, 0, dst.Num) // 解析失败留零值（宽松语义）
}

// ============ Strict（默认）+ 合法值：正常转换 ============

func TestStrictValidConversion(t *testing.T) {
	src := strictSrc{Num: "42"}
	var dst strictDst

	err := Copy(src, &dst).Do()
	assert.NoError(t, err)
	assert.Equal(t, 42, dst.Num)
}

// ============ Strict（默认）+ 非法值：返回 ErrConversionFailed ============

func TestStrictParseFailure(t *testing.T) {
	src := strictSrc{Num: "abc"}
	var dst strictDst

	err := Copy(src, &dst).Do()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConversionFailed))
}

// ============ uint / float / bool 解析失败（string 源直接拷贝） ============

func TestStrictParseFailures(t *testing.T) {
	t.Run("uint", func(t *testing.T) {
		var dst uint
		err := Copy("abc", &dst).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("float", func(t *testing.T) {
		var dst float64
		err := Copy("abc", &dst).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("bool", func(t *testing.T) {
		var dst bool
		err := Copy("abc", &dst).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})
}

// ============ map→map 值类型不兼容 ============

func TestStrictMapValueIncompatible(t *testing.T) {
	src := map[string][]int{"a": {1, 2}}

	t.Run("strict returns error", func(t *testing.T) {
		var dst map[string]string
		err := Copy(src, &dst).Do()
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("lenient silently skips key", func(t *testing.T) {
		var dst map[string]string
		err := Copy(src, &dst).Lenient().Do()
		assert.NoError(t, err)
		_, hasKey := dst["a"]
		assert.False(t, hasKey) // 不兼容值静默 continue，dst 缺该 key（宽松语义）
	})
}

// ============ strict 与 plan 缓存共存 ============

func TestStrictWithPlanCache(t *testing.T) {
	// 默认字段匹配配置（无 nameConverter 等）→ planEligible 为 true，走 plan 路径
	opt := testOpt(func(o *options) { o.strict = true })
	assert.True(t, planEligible(opt))
	assert.NotNil(t, getStructPlan(reflect.TypeOf(strictSrc{}), reflect.TypeOf(strictDst{}), opt))

	src := strictSrc{Num: "abc"}
	var dst strictDst
	err := Copy(src, &dst).Do()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConversionFailed))
}

// ============ 链式 Must() 终端在 strict 模式下 panic(ErrConversionFailed) ============

func TestStrictMustPanic(t *testing.T) {
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				err, ok := r.(error)
				assert.True(t, ok, "panic 值应为 error")
				assert.True(t, errors.Is(err, ErrConversionFailed))
			}
		}()
		// 默认 strict：解析失败 → Copy(src, &dst).Must() panic(ErrConversionFailed)
		var dst strictDst
		Copy(strictSrc{Num: "abc"}, &dst).Must()
	}()
	assert.True(t, panicked)
}
