package copier

// v0.5.1 测试：map→map 值转换 strict 报错带 key 定位（FieldPathError）、
// TypeConverter DstType 不匹配时静默回退。

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============ 修订 1：map→map 值转换 strict 报错带 key 定位 ============

func TestV051MapToMapKeyInError(t *testing.T) {
	// "Bad" 的值 []int 不可转换为 dst 元素类型 int → strict 报错，
	// 且错误经 FieldPathError 包装，Field == 对应 map key
	src := map[string]any{"Bad": []int{1}, "Ok": 1}
	var dst map[string]int

	err := Copy(src, &dst).Do()
	assert.Error(t, err)

	var fpe *FieldPathError
	assert.True(t, errors.As(err, &fpe), "错误应为 FieldPathError 包装")
	assert.Equal(t, "Bad", fpe.Field)
	assert.True(t, errors.Is(fpe.Err, ErrConversionFailed))

	// Lenient：不兼容值静默跳过，不报错
	var dst2 map[string]int
	err = Copy(src, &dst2).Lenient().Do()
	assert.NoError(t, err)
	_, hasBad := dst2["Bad"]
	assert.False(t, hasBad) // 不兼容值跳过
}

// ============ 修订 2：DstType 不匹配时转换器静默回退 ============

func TestV051TypeConvertDstTypeMismatchFallback(t *testing.T) {
	// 三元组 FieldName/SrcType 匹配但 Fn 输出类型与 DstType 不符 → 转换器不生效：
	// Fn 被调用但结果被丢弃，字段按未转换处理（原值拷贝）
	fnCalled := false
	tc := TypeConverter{
		FieldName: "Name",
		SrcType:   string(""),
		DstType:   int(0), // 声明 int，但 Fn 实际返回 string
		Fn: func(src any) (any, error) {
			fnCalled = true
			return strings.ToUpper(src.(string)), nil
		},
	}

	type s struct{ Name string }
	type d struct{ Name string }
	src := s{Name: "abc"}
	var dst d

	err := Copy(src, &dst).Converters(tc).Do()
	assert.NoError(t, err)
	assert.True(t, fnCalled)         // Fn 被调用
	assert.Equal(t, "abc", dst.Name) // 但结果被丢弃，按未转换处理
}
