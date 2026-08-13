package copier

// v0.6 测试：TypeConverter.Fn 返回 error 时 strict 模式报错
// （ErrConversionFailed 经 FieldPathError 定位，原始错误文本保留在消息中）；
// Lenient 恢复静默回退；nil 结果保持"未转换"语义。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// v06ErrConverter：匹配 "F" 字段（SrcType string），Fn 恒返回 error
func v06ErrConverter() TypeConverter {
	return TypeConverter{
		FieldName: "F",
		SrcType:   string(""),
		Fn: func(src any) (any, error) {
			return nil, errors.New("boom")
		},
	}
}

// ============ strict/Lenient/nil 语义 ============

func TestV06ConverterFnError(t *testing.T) {
	t.Run("strict reports error with field and original text", func(t *testing.T) {
		type s struct{ F string }
		type d struct{ F string }
		var dst d

		err := Copy(s{F: "x"}, &dst).Converters(v06ErrConverter()).Do()
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrConversionFailed))
		assert.Contains(t, err.Error(), "boom") // 原始错误文本保留

		var fpe *FieldPathError
		assert.True(t, errors.As(err, &fpe))
		assert.Equal(t, "F", fpe.Field)
	})

	t.Run("lenient falls back to unconverted", func(t *testing.T) {
		type s struct{ F string }
		type d struct{ F string }
		var dst d

		err := Copy(s{F: "x"}, &dst).Lenient().Converters(v06ErrConverter()).Do()
		assert.NoError(t, err)
		assert.Equal(t, "x", dst.F) // 原值进入目标（未转换处理）
	})

	t.Run("nil result stays unconverted silently", func(t *testing.T) {
		// strict 下 Fn 返回 nil：保持"未转换"语义，不报错
		tc := TypeConverter{
			FieldName: "F",
			SrcType:   string(""),
			Fn: func(src any) (any, error) {
				return nil, nil
			},
		}
		type s struct{ F string }
		type d struct{ F string }
		var dst d

		err := Copy(s{F: "x"}, &dst).Converters(tc).Do()
		assert.NoError(t, err)
		assert.Equal(t, "x", dst.F)
	})
}

// ============ 各拷贝路径错误用例 ============

func TestV06ConverterFnErrorPaths(t *testing.T) {
	t.Run("struct to struct plain path", func(t *testing.T) {
		type s struct{ F string }
		type d struct{ F string }
		var dst d
		// NameFn 使 plan 不 eligible，走逐字段路径（cpyStruct）
		err := Copy(s{F: "x"}, &dst).Converters(v06ErrConverter()).
			NameFn(func(s string) string { return s }).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("struct to struct plan path", func(t *testing.T) {
		type s struct{ F string }
		type d struct{ F string }
		// 默认配置走 plan 快路径（cpyStructByPlan）
		var dst d
		err := Copy(s{F: "x"}, &dst).Converters(v06ErrConverter()).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
		// 同类型组合再次调用命中 plan 缓存，错误仍报告
		var dst2 d
		err = Copy(s{F: "y"}, &dst2).Converters(v06ErrConverter()).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("map to map", func(t *testing.T) {
		src := map[string]any{"F": "x"}
		var dst map[string]string
		err := Copy(src, &dst).Converters(v06ErrConverter()).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
		var fpe *FieldPathError
		assert.True(t, errors.As(err, &fpe))
		assert.Equal(t, "F", fpe.Field)
	})

	t.Run("map to struct", func(t *testing.T) {
		type d struct{ F string }
		src := map[string]any{"F": "x"}
		var dst d
		err := Copy(src, &dst).Converters(v06ErrConverter()).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
		var fpe *FieldPathError
		assert.True(t, errors.As(err, &fpe))
		assert.Equal(t, "F", fpe.Field)
	})

	t.Run("struct to map", func(t *testing.T) {
		type s struct{ F string }
		src := s{F: "x"}
		var dst map[string]any
		err := Copy(src, &dst).Converters(v06ErrConverter()).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
		var fpe *FieldPathError
		assert.True(t, errors.As(err, &fpe))
		assert.Equal(t, "F", fpe.Field)
	})
}
