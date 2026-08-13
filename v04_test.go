package copier

// v0.4 新特性测试：Clone 链式化（返回 builder + Result 终端）、
// With(&Config{}) 一次性配置（非零字段覆盖语义）。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type v04CloneSrc struct {
	Name  string
	Meta  map[string]string // 零值字段（nil map）：用于验证 IgnoreEmpty 跳过
	Items []int
}

type v04StrictSrc struct{ F float64 }
type v04StrictDst struct{ F int }

type v04Node struct{ Next *v04Node }

// ============ 1. Clone 链式带选项：IgnoreEmpty 生效 + 容器深拷贝隔离 ============

func TestV04CloneChainWithOption(t *testing.T) {
	src := v04CloneSrc{Name: "x", Items: []int{1, 2}}

	got, err := Clone[v04CloneSrc](src).IgnoreEmpty().Result()
	assert.NoError(t, err)
	assert.Equal(t, "x", got.Name)

	// IgnoreEmpty 生效：src 的零值字段（nil map）被跳过，dst 保持 nil；
	// 默认行为会把 nil map 拷成非 nil 空 map，此处可区分
	assert.Nil(t, got.Meta)

	// 容器深拷贝隔离：修改副本不污染源
	got.Items[0] = 99
	assert.Equal(t, 1, src.Items[0])
}

// ============ 2. Result 终端：成功返回值；错误路径返回零值 + error ============

func TestV04ResultTerminal(t *testing.T) {
	t.Run("success returns value", func(t *testing.T) {
		src := v04CloneSrc{Name: "x", Meta: map[string]string{"k": "v"}, Items: []int{1}}
		got, err := Clone[v04CloneSrc](src).Result()
		assert.NoError(t, err)
		assert.Equal(t, src, got)

		// 深拷贝隔离
		got.Items[0] = 99
		assert.Equal(t, 1, src.Items[0])
	})

	t.Run("error path returns zero value and error", func(t *testing.T) {
		// 出错时不 panic：返回 D 零值与 error（errors.Is 可匹配哨兵）
		got, err := Clone[any](nil).Result()
		assert.True(t, errors.Is(err, ErrInvalidCopyFrom))
		assert.Nil(t, got)
	})
}

// ============ 3. With(&Config{...})：一次性配置生效 ============

func TestV04WithConfig(t *testing.T) {
	t.Run("IgnoreEmpty via Config", func(t *testing.T) {
		got, err := Clone[v04CloneSrc](v04CloneSrc{Name: "x"}).
			With(&Config{IgnoreEmpty: true}).Result()
		assert.NoError(t, err)
		assert.Nil(t, got.Meta)
	})

	t.Run("MaxDepth via Config", func(t *testing.T) {
		// 嵌套深度超过 MaxDepth 报 ErrMaxDepthExceeded（深度计数从 0 开始）
		src := &v04Node{Next: &v04Node{Next: &v04Node{}}}
		got, err := Clone[*v04Node](src).With(&Config{MaxDepth: 2}).Result()
		assert.True(t, errors.Is(err, ErrMaxDepthExceeded))
		assert.Nil(t, got) // 出错时返回 D 零值（nil 指针）
	})

	t.Run("TagName via Config", func(t *testing.T) {
		type tagSrc struct {
			Name string `custom:"toname=Renamed"`
		}
		type tagDst struct {
			Renamed string
		}
		var dst tagDst
		err := Copy(tagSrc{Name: "v"}, &dst).With(&Config{TagName: "custom"}).Do()
		assert.NoError(t, err)
		assert.Equal(t, "v", dst.Renamed)
	})
}

// ============ 4. With 空 Config{}：不改变默认行为（strict 语义保持） ============

func TestV04WithEmptyConfigKeepsDefaults(t *testing.T) {
	// strict 默认开启：float→int 精度丢失报 ErrConversionFailed。
	// 空 Config{}（全零值）不覆盖任何字段，strict 语义保持不变。
	src := v04StrictSrc{F: 3.9}
	var dst v04StrictDst
	err := Copy(src, &dst).With(&Config{}).Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))

	// MaxDepth 零值不覆盖：默认不限制深度，深层嵌套不报错
	srcN := &v04Node{Next: &v04Node{Next: &v04Node{}}}
	got, err := Clone[*v04Node](srcN).With(&Config{}).Result()
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.NotNil(t, got.Next)
}

// ============ 5. With 与链式方法混用 ============

func TestV04WithMixedWithChainMethods(t *testing.T) {
	dst := v04CloneSrc{Name: "keep", Meta: map[string]string{"keep": "1"}}
	err := Copy(v04CloneSrc{Name: "x"}, &dst).
		IgnoreEmpty().                          // 链式方法
		With(&Config{CaseSensitive: true}).Do() // With 配置
	assert.NoError(t, err)

	// IgnoreEmpty（链式）生效：src 的 nil map 字段被跳过 → dst.Meta 保持原值
	assert.Equal(t, map[string]string{"keep": "1"}, dst.Meta)
	assert.Equal(t, "x", dst.Name)
}

// ============ 6. Copy 入口调用 Result()：返回 dst 值副本（无害性） ============

func TestV04CopyResultHarmless(t *testing.T) {
	dst := v04CloneSrc{Name: "pre"}
	got, err := Copy(v04CloneSrc{Name: "x", Items: []int{1}}, &dst).Result()
	assert.NoError(t, err)

	// Result 返回 dst 当前值副本，与 dst 内容一致；外部 dst 仍被正常填充
	assert.Equal(t, "x", got.Name)
	assert.Equal(t, dst, got)
	assert.Equal(t, "x", dst.Name)

	// 返回值为值副本：修改 got 不影响外部 dst（D 为值类型时）
	got.Name = "mut"
	assert.Equal(t, "x", dst.Name)
}
