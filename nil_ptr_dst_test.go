package copier

// 顶层 nil 指针 dst 自动 New：Copy(&p, src)（p 为 nil *T）自动分配指针链，
// 语义与 deepCopyInner Pointer 分支自动 New 对齐；nil 指针值作为 any 传入
// （Copy(p, src)）时反射层不可设置，维持原 ErrInvalidCopyDestination 行为。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type npSrc struct {
	Name string
	Age  int
}

func TestNilPtrDstAutoNew(t *testing.T) {
	src := npSrc{Name: "n", Age: 1}

	t.Run("Copy(&p, src) allocates nil pointer", func(t *testing.T) {
		var p *npSrc
		err := Copy(&p, src)
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "n", p.Name)
		assert.Equal(t, 1, p.Age)
	})

	t.Run("multi-level nil pointer chain", func(t *testing.T) {
		var pp **npSrc
		err := Copy(&pp, src)
		assert.NoError(t, err)
		assert.NotNil(t, pp)
		assert.NotNil(t, *pp)
		assert.Equal(t, "n", (*pp).Name)
	})

	t.Run("value dst unchanged", func(t *testing.T) {
		var v npSrc
		err := Copy(&v, src)
		assert.NoError(t, err)
		assert.Equal(t, "n", v.Name)
	})

	t.Run("nil pointer value as dst still errors", func(t *testing.T) {
		var p *npSrc
		err := Copy(p, src)
		assert.True(t, errors.Is(err, ErrInvalidCopyDestination))
	})
}

// 嵌套指针字段自动 New 回归（deepCopyInner Pointer 分支，既有行为）。
type npOuter struct {
	Ptr *npSrc
}

func TestNilPtrNestedFieldAutoNew(t *testing.T) {
	src := npOuter{Ptr: &npSrc{Name: "n"}}
	var dst npOuter

	err := Copy(&dst, src)
	assert.NoError(t, err)
	assert.NotNil(t, dst.Ptr)
	assert.Equal(t, "n", dst.Ptr.Name)

	// nil 源指针 → dst 置 nil
	src2 := npOuter{}
	var dst2 npOuter
	dst2.Ptr = &npSrc{}
	err = Copy(&dst2, src2)
	assert.NoError(t, err)
	assert.Nil(t, dst2.Ptr)
}
