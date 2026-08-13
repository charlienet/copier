package copier

// v0.3 链式泛型 API 测试：类型推导、13 个链式方法行为断言、
// 错误字段路径、多链式组合、Do 幂等复用。

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type chSrc struct {
	Name string
	Age  int
	Tags []string
	Must string `copier:"must"`
}

type chDst struct {
	Name string
	Age  int
	Tags []string
	Must string
}

type chSetterDst struct {
	Stored string
}

func (d *chSetterDst) Name(v string) { d.Stored = v }

// ============ a) 编译期类型推导（不显式写 [S,R]） ============

func TestCopyTypeInference(t *testing.T) {
	src := chSrc{Name: "n"}
	var dst chDst

	b := Copy(src, &dst) // S/R 由参数推导
	assert.NotNil(t, b)
	assert.NoError(t, b.Do())
	assert.Equal(t, "n", dst.Name)

	// 字面量推导
	var dst2 chDst
	assert.NoError(t, Copy(chSrc{Name: "x"}, &dst2).Do())
	assert.Equal(t, "x", dst2.Name)
}

// ============ b) 链式方法行为断言（v0.3 起 With* 已删除，直接验证行为） ============

func TestCopyEquivalence(t *testing.T) {
	t.Run("Lenient", func(t *testing.T) {
		type s struct{ Num string }
		type d struct{ Num int }
		src := s{Num: "abc"}

		var chainDst d
		assert.NoError(t, Copy(src, &chainDst).Lenient().Do())
	})

	t.Run("IgnoreEmpty", func(t *testing.T) {
		src := chSrc{Name: "n"} // Age/Tags/Must 零值

		var chainDst chDst
		assert.NoError(t, Copy(src, &chainDst).IgnoreEmpty().Do())
	})

	t.Run("CaseSensitive", func(t *testing.T) {
		type srcT struct{ UserID string }
		type dstT struct {
			userid string // 未导出，声明在前
			UserID string
		}
		src := srcT{UserID: "u"}

		// 不敏感：匹配未导出 userid → 跳过
		var l1 dstT
		assert.NoError(t, Copy(src, &l1).Do())
		assert.Equal(t, "", l1.UserID)

		// 敏感（链式 vs 直调）：精确匹配 UserID
		var c1, c2 dstT
		assert.NoError(t, Copy(src, &c1).CaseSensitive().Do())
		assert.NoError(t, Copy(src, &c2).CaseSensitive().Do())
		assert.Equal(t, c2, c1)
		assert.Equal(t, "u", c1.UserID)
	})

	t.Run("MustFields", func(t *testing.T) {
		src := chSrc{Name: "n", Must: "m"}

		var chainDst chDst
		assert.NoError(t, Copy(src, &chainDst).MustFields().Do())
		assert.Equal(t, "", chainDst.Name) // 非 must 字段不被拷贝
		assert.Equal(t, "m", chainDst.Must)
	})

	t.Run("SkipFields", func(t *testing.T) {
		src := chSrc{Name: "n", Age: 1}

		var chainDst chDst
		assert.NoError(t, Copy(src, &chainDst).SkipFields("Name").Do())
		assert.Equal(t, "", chainDst.Name)
	})

	t.Run("MaxDepth", func(t *testing.T) {
		type inner struct{ N int }
		type s struct{ In inner }
		type d struct{ In inner }
		src := s{In: inner{N: 1}}

		var chainDst d
		err1 := Copy(src, &chainDst).MaxDepth(0).Do()
		assert.True(t, errors.Is(err1, ErrMaxDepthExceeded))
	})

	t.Run("NilSrcZero", func(t *testing.T) {
		var chainDst chDst
		chainDst.Age = 42

		// S 显式指定（nil 无法推导）
		assert.NoError(t, Copy[*chSrc](nil, &chainDst).NilSrcZero().Do())
		assert.Equal(t, 0, chainDst.Age) // 置零
	})

	t.Run("TagName", func(t *testing.T) {
		type s struct {
			Name string `json:"toname=json_name"`
		}
		type d struct {
			Json_Name string
		}
		src := s{Name: "n"}

		var chainDst d
		assert.NoError(t, Copy(src, &chainDst).TagName("json").Do())
		assert.Equal(t, "n", chainDst.Json_Name)
	})

	t.Run("Converters", func(t *testing.T) {
		// TypeConvert 全路径生效（v0.4.1：struct→struct / struct→map / map→struct / map→map），用 map 目标验证
		type s struct{ Name string }
		src := s{Name: "abc"}
		tc := TypeConverter{
			FieldName: "Name",
			SrcType:   string(""),
			DstType:   int(0),
			Fn: func(src any) (any, error) {
				return len(src.(string)), nil
			},
		}

		var chainDst map[string]any
		assert.NoError(t, Copy(src, &chainDst).Converters(tc).Do())
		assert.Equal(t, 3, chainDst["Name"])
	})

	t.Run("ValueConverter", func(t *testing.T) {
		var chainNames []string
		src := chSrc{Name: "n", Age: 1}

		var chainDst chDst
		vc := func(name string, v any) any {
			chainNames = append(chainNames, name)
			return v
		}
		assert.NoError(t, Copy(src, &chainDst).ValueConverter(vc).Do())
		assert.Contains(t, chainNames, "Name")
		assert.Contains(t, chainNames, "Age")
	})

	t.Run("MethodMapping", func(t *testing.T) {
		src := chSrc{Name: "x"}

		var chainDst chSetterDst
		assert.NoError(t, Copy(src, &chainDst).MethodMapping().Do())
		assert.Equal(t, "x", chainDst.Stored)
	})

	t.Run("NameFn", func(t *testing.T) {
		type s struct{ UserId string }
		type d struct{ UserID string }
		src := s{UserId: "u"}

		var chainDst d
		fn := func(n string) string {
			if n == "UserId" {
				return "UserID"
			}
			return n
		}
		assert.NoError(t, Copy(src, &chainDst).NameFn(fn).Do())
		assert.Equal(t, "u", chainDst.UserID)
	})

	t.Run("NameMapping", func(t *testing.T) {
		type s struct{ UserId string }
		type d struct{ UserID string }
		src := s{UserId: "u"}

		var chainDst d
		m := map[string]string{"UserId": "UserID"}
		assert.NoError(t, Copy(src, &chainDst).NameMapping(m).Do())
		assert.Equal(t, "u", chainDst.UserID)
	})

	t.Run("default equals Copy (strict on)", func(t *testing.T) {
		src := chSrc{Name: "n", Age: 1}
		var chainDst chDst
		assert.NoError(t, Copy(src, &chainDst).Do())
	})
}

// ============ c) 错误链：strict 下解析失败含字段名 ============

func TestCopyErrorFieldPath(t *testing.T) {
	type pSrc struct{ Title string }
	type pDst struct{ Title int }
	src := pSrc{Title: "abc"}
	var dst pDst

	err := Copy(src, &dst).Do() // 默认 strict
	assert.True(t, errors.Is(err, ErrConversionFailed))
	assert.Contains(t, err.Error(), "Title:")
}

// ============ d) 多链式组合 ============

func TestCopyChainCombination(t *testing.T) {
	type inner struct{ N int }
	type s struct {
		Name  string
		Age   int
		In    inner
		Other string
	}
	type d struct {
		name  string // 未导出，声明在前
		Name  string
		Age   int
		In    inner
		Other string
	}
	src := s{Name: "n", Age: 1, In: inner{N: 5}, Other: "o"}
	var dst d

	err := Copy(src, &dst).IgnoreEmpty().CaseSensitive().MaxDepth(3).Do()
	assert.NoError(t, err)
	assert.Equal(t, "n", dst.Name) // caseSensitive 精确匹配导出字段
	assert.Equal(t, 1, dst.Age)    // 非零值写入
	assert.Equal(t, 5, dst.In.N)   // 嵌套 struct 正常
	assert.Equal(t, "o", dst.Other)
}

// ============ e) Do 幂等/复用 ============

func TestCopyDoReuse(t *testing.T) {
	src := chSrc{Name: "n", Age: 1}
	var dst chDst

	b := Copy(src, &dst)
	assert.NoError(t, b.Do())
	first := dst

	assert.NoError(t, b.Do()) // 同一实例二次执行
	assert.Equal(t, first, dst)
	assert.Equal(t, "n", dst.Name)
}

func TestCopyDoReuseWithOptions(t *testing.T) {
	// 带选项的 builder 复用：同一实例重复 Do() 每次应用同一组选项，结果幂等
	src := chSrc{Name: "n", Age: 0} // Age 零值：IgnoreEmpty 跳过
	var dst chDst

	b := Copy(src, &dst).IgnoreEmpty().CaseSensitive()
	assert.NoError(t, b.Do())
	first := dst
	assert.Equal(t, "n", dst.Name)
	assert.Equal(t, 0, dst.Age) // Age 零值被忽略

	dst = chDst{} // 重置目标后再执行
	assert.NoError(t, b.Do())
	assert.Equal(t, first, dst) // 两次执行结果一致（幂等）
	assert.Equal(t, "n", dst.Name)
}

// ============ e2) 终端错误处理：正常路径 + 错误路径 + Lenient 恢复 ============

func TestCopyErrorHandling(t *testing.T) {
	t.Run("normal path fills dst with no error", func(t *testing.T) {
		src := chSrc{Name: "n", Age: 1}
		var dst chDst

		err := Copy(src, &dst).Do()
		assert.NoError(t, err)
		assert.Equal(t, "n", dst.Name)
		assert.Equal(t, 1, dst.Age)
	})

	t.Run("error on strict parse failure", func(t *testing.T) {
		type s struct{ Num string }
		type d struct{ Num int }
		src := s{Num: "abc"}

		var dst d
		err := Copy(src, &dst).Do()
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrConversionFailed))
	})

	t.Run("Lenient().Do() succeeds", func(t *testing.T) {
		type s struct{ Num string }
		type d struct{ Num int }
		src := s{Num: "abc"}

		var dst d
		err := Copy(src, &dst).Lenient().Do() // 宽松模式解析失败静默，不报错
		assert.NoError(t, err)
		assert.Equal(t, 0, dst.Num)
	})
}

// ============ f) benchmark：Copy 链式入口 ============

func BenchmarkCopyNoOpt(b *testing.B) {
	src := benchMedium{
		Name: "John", Age: 30, Height: 1.75, Active: true,
		Tags: []string{"a", "b"}, Meta: map[string]string{"k": "v"},
		City: "Beijing", Score: 100, Level: 3, Created: time.Now(),
	}
	var dst benchMedium

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := Copy(src, &dst).Do(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopy5Opts(b *testing.B) {
	src := benchMedium{
		Name: "John", Age: 30, Height: 1.75, Active: true,
		Tags: []string{"a", "b"}, Meta: map[string]string{"k": "v"},
		City: "Beijing", Score: 100, Level: 3, Created: time.Now(),
	}
	var dst benchMedium

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := Copy(src, &dst).
			IgnoreEmpty().CaseSensitive().MaxDepth(3).SkipFields("X").TagName("copier").Do(); err != nil {
			b.Fatal(err)
		}
	}
}
