package copier

// v0.5.0 测试：AllowPrecisionLoss（strict 精度豁免）、FieldPathError（结构化
// 字段路径错误）、Config.Strict / Strict()（Lenient 双向闭环）。

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============ 1. AllowPrecisionLoss：strict 下仅豁免数值精度检查 ============

func TestV05AllowPrecisionLoss(t *testing.T) {
	type fsrc struct{ N float64 }
	type fdst struct{ N int }
	src := fsrc{N: 3.9}

	t.Run("strict rejects precision loss by default", func(t *testing.T) {
		var dst fdst
		err := Copy(src, &dst).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("AllowPrecisionLoss truncates instead of erroring", func(t *testing.T) {
		var dst fdst
		err := Copy(src, &dst).AllowPrecisionLoss().Do()
		assert.NoError(t, err)
		assert.Equal(t, 3, dst.N)
	})

	t.Run("parse failures still error under AllowPrecisionLoss", func(t *testing.T) {
		// 与 Lenient 正交：仅豁免精度类，字符串解析失败仍报 ErrConversionFailed
		type ssrc struct{ Num string }
		type sdst struct{ Num int }
		var dst sdst
		err := Copy(ssrc{Num: "abc"}, &dst).AllowPrecisionLoss().Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("Config.AllowPrecisionLoss applies", func(t *testing.T) {
		var dst fdst
		err := Copy(src, &dst).With(&Config{AllowPrecisionLoss: true}).Do()
		assert.NoError(t, err)
		assert.Equal(t, 3, dst.N)
	})

	t.Run("plan path no regression", func(t *testing.T) {
		// struct→struct 默认配置走 plan 缓存路径，AllowPrecisionLoss 同样生效
		type psrc struct{ N float64 }
		type pdst struct{ N int }
		var dst pdst
		err := Copy(psrc{N: 3.9}, &dst).AllowPrecisionLoss().Do()
		assert.NoError(t, err)
		assert.Equal(t, 3, dst.N)
	})
}

// ============ 2. FieldPathError：结构化字段路径错误 ============

func TestV05FieldPathError(t *testing.T) {
	t.Run("Error string matches fmt wrapping byte-for-byte", func(t *testing.T) {
		want := fmt.Errorf("%s: %w", "F", ErrConversionFailed).Error()
		fpe := &FieldPathError{Field: "F", Err: ErrConversionFailed}
		assert.Equal(t, want, fpe.Error())
		assert.True(t, errors.Is(fpe, ErrConversionFailed)) // Unwrap 链直达哨兵
	})

	t.Run("errors.Is reaches sentinel through nested chain", func(t *testing.T) {
		type inner struct{ N int }
		type outer struct{ Inner inner }
		type innerS struct{ N string }
		type outerS struct{ Inner innerS }

		var dst outer
		err := Copy(outerS{Inner: innerS{N: "abc"}}, &dst).Do()
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("nested path collected via Unwrap chain", func(t *testing.T) {
		type inner struct{ N int }
		type outer struct{ Inner inner }
		type innerS struct{ N string }
		type outerS struct{ Inner innerS }

		var dst outer
		err := Copy(outerS{Inner: innerS{N: "abc"}}, &dst).Do()

		// 沿 Unwrap 链收集所有 FieldPathError：外层先出现，内层后出现
		var fields []string
		for e := err; e != nil; {
			var fpe *FieldPathError
			if !errors.As(e, &fpe) {
				break
			}
			fields = append(fields, fpe.Field)
			e = fpe.Err
		}
		assert.Equal(t, []string{"Inner", "N"}, fields) // 字段路径完整

		// 错误字符串含完整嵌套路径
		assert.Contains(t, err.Error(), "Inner: N: ")
	})
}

// ============ 3. Config.Strict / Strict()：Lenient 双向闭环 ============

func TestV05StrictRoundTrip(t *testing.T) {
	type ssrc struct{ Num string }
	type sdst struct{ Num int }
	src := ssrc{Num: "abc"}

	t.Run("Lenient then With(Strict) restores strict", func(t *testing.T) {
		var dst sdst
		err := Copy(src, &dst).Lenient().With(&Config{Strict: true}).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("Lenient and Strict both set: Strict wins", func(t *testing.T) {
		var dst sdst
		err := Copy(src, &dst).With(&Config{Lenient: true, Strict: true}).Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("Strict zero value does not override", func(t *testing.T) {
		// Lenient() 后 With(&Config{})：Strict 零值不构成"设置" → 仍宽松
		var dst sdst
		err := Copy(src, &dst).Lenient().With(&Config{}).Do()
		assert.NoError(t, err)
		assert.Equal(t, 0, dst.Num)
	})

	t.Run("chain Strict() restores strict", func(t *testing.T) {
		var dst sdst
		err := Copy(src, &dst).Lenient().Strict().Do()
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})
}
