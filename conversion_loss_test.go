package copier

// 严格模式数值精度丢失拦截：WithStrict 下 float→int 截断/溢出、
// float64→float32 舍入、int→float 超精确范围均返回 ErrConversionFailed；
// 默认宽松模式零变化。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrictPrecisionLoss(t *testing.T) {
	t.Run("float64 3.7 to int errors", func(t *testing.T) {
		var dst int
		err := Copy(&dst, 3.7, WithStrict())
		assert.True(t, errors.Is(err, ErrConversionFailed))
		assert.Contains(t, err.Error(), "precision loss")
	})

	t.Run("float64 3.0 to int ok", func(t *testing.T) {
		var dst int
		err := Copy(&dst, 3.0, WithStrict())
		assert.NoError(t, err)
		assert.Equal(t, 3, dst)
	})

	t.Run("lenient truncates silently", func(t *testing.T) {
		var dst int
		err := Copy(&dst, 3.7, WithLenient())
		assert.NoError(t, err)
		assert.Equal(t, 3, dst)
	})

	t.Run("float64 to float32 precision loss", func(t *testing.T) {
		var dst float32
		err := Copy(&dst, 0.1, WithStrict())
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("float64 3.5 to float32 lossless", func(t *testing.T) {
		var dst float32
		err := Copy(&dst, 3.5, WithStrict())
		assert.NoError(t, err)
		assert.Equal(t, float32(3.5), dst)
	})

	t.Run("large int64 beyond float64 precision errors", func(t *testing.T) {
		var dst float64
		err := Copy(&dst, int64(9007199254740993), WithStrict())
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("small int to float lossless", func(t *testing.T) {
		var dst float64
		err := Copy(&dst, 42, WithStrict())
		assert.NoError(t, err)
		assert.Equal(t, float64(42), dst)
	})

	t.Run("struct field precision loss via plan path", func(t *testing.T) {
		type s struct{ F float64 }
		type d struct{ F int }

		src := s{F: 3.7}
		var dst d
		err := Copy(&dst, src, WithStrict())
		assert.True(t, errors.Is(err, ErrConversionFailed))
		// 字段路径（②）与精度检查（⑦）叠加：字段名 + 精度丢失
		assert.Contains(t, err.Error(), "F:")
		assert.Contains(t, err.Error(), "precision loss")
	})

	t.Run("int to int not checked", func(t *testing.T) {
		// 整数→整数（含溢出）不在本次检查范围：严格模式仍静默转换
		var dst int8
		err := Copy(&dst, 300, WithStrict())
		assert.NoError(t, err)
		_ = dst
	})
}
