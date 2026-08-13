package copier

// v0.4.1 测试：TypeConverter 全路径修复——struct→struct / map→struct 路径
// 补 TypeConvert 调用，注册的转换器不再静默失效。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type v041StructSrc struct {
	Name string
	Age  int
}

// v041StructDst：Age 用 int64（与转换器输出同型），Name 与 src 同型
type v041StructDst struct {
	Name string
	Age  int64
}

// v041AgeX10：Age int→int64 ×10 转换器（FieldName/SrcType/DstType 三重匹配）
func v041AgeX10() TypeConverter {
	return TypeConverter{
		FieldName: "Age",
		SrcType:   int(0),
		DstType:   int64(0),
		Fn: func(src any) (any, error) {
			return int64(src.(int) * 10), nil
		},
	}
}

// ============ ① struct→struct 生效（普通路径 + plan 缓存路径） ============

func TestV041TypeConvertStructToStruct(t *testing.T) {
	// 普通路径（逐字段循环）：NameFn 使 plan 不 eligible
	var dst1 v041StructDst
	err := Copy(v041StructSrc{Name: "n", Age: 7}, &dst1).
		Converters(v041AgeX10()).NameFn(func(s string) string { return s }).Do()
	assert.NoError(t, err)
	assert.Equal(t, int64(70), dst1.Age)
	assert.Equal(t, "n", dst1.Name)

	// plan 缓存路径：默认配置走 plan，转换器同样生效
	var dst2 v041StructDst
	err = Copy(v041StructSrc{Name: "n", Age: 7}, &dst2).Converters(v041AgeX10()).Do()
	assert.NoError(t, err)
	assert.Equal(t, int64(70), dst2.Age)

	// 同一类型组合再次调用命中 plan 缓存，转换仍生效
	var dst3 v041StructDst
	err = Copy(v041StructSrc{Name: "n", Age: 8}, &dst3).Converters(v041AgeX10()).Do()
	assert.NoError(t, err)
	assert.Equal(t, int64(80), dst3.Age)
}

// ============ ② map→struct 生效（普通 + plan 路径） ============

func TestV041TypeConvertMapToStruct(t *testing.T) {
	srcMap := map[string]any{"Name": "n", "Age": 7}

	// 普通路径：skipFields 使 plan 不 eligible（字段名不实际存在于 src/dst）
	var dst1 v041StructDst
	err := Copy(srcMap, &dst1).Converters(v041AgeX10()).SkipFields("Nonexistent").Do()
	assert.NoError(t, err)
	assert.Equal(t, int64(70), dst1.Age)
	assert.Equal(t, "n", dst1.Name)

	// plan 路径：默认配置走 cpyMapToStruct
	var dst2 v041StructDst
	err = Copy(srcMap, &dst2).Converters(v041AgeX10()).Do()
	assert.NoError(t, err)
	assert.Equal(t, int64(70), dst2.Age)
	assert.Equal(t, "n", dst2.Name)
}

// ============ ③ 三重匹配不匹配时降级为原值 ============

func TestV041TypeConvertNoMatchFallsBack(t *testing.T) {
	t.Run("src type mismatch", func(t *testing.T) {
		// FieldName 匹配但 SrcType 不匹配（Age 是 int，这里声明 string）→ 转换器不适用
		tc := TypeConverter{
			FieldName: "Age",
			SrcType:   string(""),
			Fn: func(src any) (any, error) {
				return nil, errors.New("should not be called")
			},
		}
		var dst v041StructDst
		err := Copy(v041StructSrc{Name: "n", Age: 7}, &dst).Converters(tc).Do()
		assert.NoError(t, err)
		assert.Equal(t, int64(7), dst.Age) // 原值经自动转换拷贝
	})

	t.Run("field name mismatch", func(t *testing.T) {
		tc := TypeConverter{
			FieldName: "Other",
			SrcType:   int(0),
			Fn: func(src any) (any, error) {
				return nil, errors.New("should not be called")
			},
		}
		var dst v041StructDst
		err := Copy(v041StructSrc{Name: "n", Age: 7}, &dst).Converters(tc).Do()
		assert.NoError(t, err)
		assert.Equal(t, int64(7), dst.Age)
	})
}

// ============ ④ converter 输出与 dst 字段类型不匹配：strict 报 ErrConversionFailed ============

func TestV041TypeConvertStrictMismatch(t *testing.T) {
	// converter 输出 float64 到 int64 字段：有损转换，strict 精度检查报错
	tc := TypeConverter{
		FieldName: "Age",
		SrcType:   int(0),
		Fn: func(src any) (any, error) {
			return 3.9, nil // float64
		},
	}

	// strict（默认）：ErrConversionFailed
	var dst v041StructDst
	err := Copy(v041StructSrc{Name: "n", Age: 7}, &dst).Converters(tc).Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))

	// Lenient：截断不报错（3.9 → 3）
	var dst2 v041StructDst
	err = Copy(v041StructSrc{Name: "n", Age: 7}, &dst2).Converters(tc).Lenient().Do()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), dst2.Age)
}

// ============ ⑤ converter 输出与 dst 字段精确同类型时无损通过 ============

func TestV041TypeConvertExactTypeNoPrecisionCheck(t *testing.T) {
	// converter 输出 int64 与 dst.Age (int64) 精确同型：不触发精度检查，大值也无损通过
	tc := TypeConverter{
		FieldName: "Age",
		SrcType:   int(0),
		DstType:   int64(0),
		Fn: func(src any) (any, error) {
			return int64(src.(int) * 1000), nil
		},
	}

	var dst v041StructDst
	err := Copy(v041StructSrc{Name: "n", Age: 7}, &dst).Converters(tc).Do()
	assert.NoError(t, err)
	assert.Equal(t, int64(7000), dst.Age)
}
