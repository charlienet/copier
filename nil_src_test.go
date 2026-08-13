package copier

// NilSrcZero：nil 源视为零值目标（Copy[*nsSrc](nil, &dst).Do() 置零返回 nil），
// 默认关闭（nil 源返回 ErrInvalidCopyFrom）。与 strict 正交，不受其影响。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type nsSrc struct{ V int }

func TestNilSrcZero(t *testing.T) {
	t.Run("default nil src errors", func(t *testing.T) {
		var dst nsSrc
		err := Copy[*nsSrc](nil, &dst).Do()
		assert.True(t, errors.Is(err, ErrInvalidCopyFrom))
	})

	t.Run("NilSrcZero zeroes dst", func(t *testing.T) {
		dst := nsSrc{V: 42}
		err := Copy[*nsSrc](nil, &dst).NilSrcZero().Do()
		assert.NoError(t, err)
		assert.Equal(t, nsSrc{}, dst) // 原值 42 → 0
	})

	t.Run("NilSrcZero unaffected by default strict", func(t *testing.T) {
		// 默认 strict=true 已开启，nil 源置零仍成功（正交）
		dst := nsSrc{V: 1}
		err := Copy[*nsSrc](nil, &dst).NilSrcZero().Do()
		assert.NoError(t, err)
		assert.Equal(t, nsSrc{}, dst)
	})

	t.Run("NilSrcZero normal src unchanged", func(t *testing.T) {
		dst := nsSrc{}
		err := Copy(nsSrc{V: 7}, &dst).NilSrcZero().Do()
		assert.NoError(t, err)
		assert.Equal(t, 7, dst.V)
	})

	t.Run("NilSrcZero zeroes map dst", func(t *testing.T) {
		dst := map[string]any{"keep": 1}
		err := Copy[*nsSrc](nil, &dst).NilSrcZero().Do()
		assert.NoError(t, err)
		assert.Nil(t, dst) // map 置零
	})

	t.Run("nil ptr dst allocated and zeroed", func(t *testing.T) {
		var p *nsSrc
		err := Copy[*nsSrc](nil, &p).NilSrcZero().Do()
		assert.NoError(t, err)
		// 顶层分配（阶段 A）+ nil 源置零：p 非 nil，指向零值
		assert.NotNil(t, p)
		assert.Equal(t, nsSrc{}, *p)
	})

	t.Run("generic Clone keeps ErrInvalidCopyFrom for nil src", func(t *testing.T) {
		// Clone 不接收选项：nil 源仍走默认 nilSrcZero=false 语义
		_, err := Clone[any](nil)
		assert.True(t, errors.Is(err, ErrInvalidCopyFrom))
	})
}
