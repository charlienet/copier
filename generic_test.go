package copier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type gCloneInner struct{ N int }

type gCloneSrc struct {
	Name  string
	Inner gCloneInner
	Items []int
	Meta  map[string]string
}

// ============ Clone：值类型 struct（含嵌套/切片/map 字段） ============

func TestGenericCloneStruct(t *testing.T) {
	src := gCloneSrc{
		Name:  "x",
		Inner: gCloneInner{N: 1},
		Items: []int{1, 2},
		Meta:  map[string]string{"k": "v"},
	}

	got, err := Clone(src)
	assert.NoError(t, err)
	assert.Equal(t, src, got)

	// 深拷贝隔离：修改副本不污染源
	got.Items[0] = 99
	got.Meta["k"] = "zzz"
	got.Inner.N = 99
	assert.Equal(t, 1, src.Items[0])
	assert.Equal(t, "v", src.Meta["k"])
	assert.Equal(t, 1, src.Inner.N)
}

// ============ Clone：指针类型（T=*Foo） ============

func TestGenericClonePointer(t *testing.T) {
	t.Run("valid pointer deep copied", func(t *testing.T) {
		p := &gCloneSrc{Name: "x", Items: []int{1}}

		got, err := Clone(p)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, "x", got.Name)

		got.Items[0] = 99
		assert.Equal(t, 1, p.Items[0])
	})

	t.Run("nil pointer returns ErrInvalidCopyFrom", func(t *testing.T) {
		var p *gCloneSrc

		got, err := Clone(p)
		assert.True(t, errors.Is(err, ErrInvalidCopyFrom))
		assert.Nil(t, got) // 出错时返回 T 零值（nil 指针）
	})
}

// ============ Clone：map / slice 类型 ============

func TestGenericCloneMap(t *testing.T) {
	src := map[string]any{"a": 1, "b": []int{1, 2}}

	got, err := Clone(src)
	assert.NoError(t, err)
	assert.Equal(t, src, got)

	// 深拷贝隔离（嵌套 slice 独立副本）
	got["b"].([]int)[0] = 99
	assert.Equal(t, 1, src["b"].([]int)[0])
}

func TestGenericCloneSlice(t *testing.T) {
	src := []int{1, 2, 3}

	got, err := Clone(src)
	assert.NoError(t, err)
	assert.Equal(t, src, got)

	got[0] = 99
	assert.Equal(t, 1, src[0])
}

// ============ MustClone：成功 + panic 路径 ============

func TestGenericMustClone(t *testing.T) {
	got := MustClone(gCloneSrc{Name: "x"})
	assert.Equal(t, "x", got.Name)
}

func TestGenericMustClonePanic(t *testing.T) {
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				err, ok := r.(error)
				assert.True(t, ok, "panic 值应为 error")
				assert.True(t, errors.Is(err, ErrInvalidCopyFrom))
			}
		}()
		// T=any 且 src=nil → from.IsValid()==false → ErrInvalidCopyFrom
		MustClone[any](nil)
	}()
	assert.True(t, panicked)
}

// ============ Convert：跨类型 struct → struct ============

type gConvSrc struct {
	Name string
	Age  int
}

type gConvDst struct {
	Name string
	Age  int64 // 类型不同（int→int64），验证自动转换
}

func TestGenericConvert(t *testing.T) {
	got, err := Convert[gConvSrc, gConvDst](gConvSrc{Name: "n", Age: 30})
	assert.NoError(t, err)
	assert.Equal(t, "n", got.Name)
	assert.Equal(t, int64(30), got.Age)
}

// ============ Convert：到 interface 类型（D=any） ============

func TestGenericConvertToInterface(t *testing.T) {
	src := map[string]any{"a": 1}

	got, err := Convert[map[string]any, any](src)
	assert.NoError(t, err)
	gotMap, ok := got.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, src, gotMap)

	// interface 承载深拷贝副本：修改副本不污染源
	gotMap["a"] = 99
	assert.Equal(t, 1, src["a"])
}

// ============ MustConvert：成功 + panic 路径 ============

func TestGenericMustConvert(t *testing.T) {
	got := MustConvert[gConvSrc, gConvDst](gConvSrc{Name: "n", Age: 1})
	assert.Equal(t, int64(1), got.Age)
	assert.Equal(t, "n", got.Name)
}

func TestGenericMustConvertPanic(t *testing.T) {
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				err, ok := r.(error)
				assert.True(t, ok, "panic 值应为 error")
				assert.True(t, errors.Is(err, ErrInvalidCopyFrom))
			}
		}()
		MustConvert[any, any](nil)
	}()
	assert.True(t, panicked)
}
