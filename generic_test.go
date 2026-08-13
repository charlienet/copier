package copier

// 泛型 API 测试：Clone 同类型深拷贝；跨类型走 Copy；panic 统一到 Copy(...).Must()
// Clone[T] 同类型深拷贝；跨类型（含 struct→map）走 Copy。

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

// ============ Clone[T]：值类型 struct（含嵌套/切片/map 字段） ============

func TestGenericCloneSameTypeStruct(t *testing.T) {
	src := gCloneSrc{
		Name:  "x",
		Inner: gCloneInner{N: 1},
		Items: []int{1, 2},
		Meta:  map[string]string{"k": "v"},
	}

	got, err := Clone[gCloneSrc](src)
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

// ============ Clone[T]：指针类型（S=*Foo） ============

func TestGenericCloneSameTypePointer(t *testing.T) {
	t.Run("valid pointer deep copied", func(t *testing.T) {
		p := &gCloneSrc{Name: "x", Items: []int{1}}

		got, err := Clone[*gCloneSrc](p)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, "x", got.Name)

		got.Items[0] = 99
		assert.Equal(t, 1, p.Items[0])
	})

	t.Run("nil pointer returns ErrInvalidCopyFrom", func(t *testing.T) {
		var p *gCloneSrc

		got, err := Clone[*gCloneSrc](p)
		assert.True(t, errors.Is(err, ErrInvalidCopyFrom))
		assert.Nil(t, got) // 出错时返回 D 零值（nil 指针）
	})
}

// ============ Clone[T]：map / slice 类型 ============

func TestGenericCloneSameTypeMap(t *testing.T) {
	src := map[string]any{"a": 1, "b": []int{1, 2}}

	got, err := Clone[map[string]any](src)
	assert.NoError(t, err)
	assert.Equal(t, src, got)

	// 深拷贝隔离（嵌套 slice 独立副本）
	got["b"].([]int)[0] = 99
	assert.Equal(t, 1, src["b"].([]int)[0])
}

func TestGenericCloneSameTypeSlice(t *testing.T) {
	src := []int{1, 2, 3}

	got, err := Clone[[]int](src)
	assert.NoError(t, err)
	assert.Equal(t, src, got)

	got[0] = 99
	assert.Equal(t, 1, src[0])
}

// panic 语义统一到 Copy(...).Must() 终端：
// nil 源场景经 Copy[any](nil, &dst).Must() panic(ErrInvalidCopyFrom)。
func TestGenericMustPanicOnNilSrc(t *testing.T) {
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
		// src=nil（S=any）→ from.IsValid()==false → ErrInvalidCopyFrom → Must() panic
		var dst any
		Copy[any](nil, &dst).Must()
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

func TestGenericCloneCrossType(t *testing.T) {
	// 跨类型（Clone 仅同类型）走 Copy：src struct → dst struct（int→int64 自动转换）
	var dst gConvDst
	err := Copy(gConvSrc{Name: "n", Age: 30}, &dst).Do()
	assert.NoError(t, err)
	assert.Equal(t, "n", dst.Name)
	assert.Equal(t, int64(30), dst.Age)
}

// ============ 跨类型到 interface（D=any，走 Copy） ============

func TestGenericCloneToInterface(t *testing.T) {
	src := map[string]any{"a": 1}

	var got any
	assert.NoError(t, Copy(src, &got).Do())
	gotMap, ok := got.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, src, gotMap)

	// interface 承载深拷贝副本：修改副本不污染源
	gotMap["a"] = 99
	assert.Equal(t, 1, src["a"])
}

// 跨类型场景（Clone 仅同类型）用 Copy 验证，见 TestGenericCloneCrossType。
