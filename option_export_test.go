package copier_test

// 外部视角测试：验证导出类型 copier.Option 的可见性，
// 使封装层可用显式签名 ...copier.Option 透传选项（无需 reflect 转发）。

import (
	"testing"

	"github.com/charlienet/copier"
	"github.com/stretchr/testify/assert"
)

// copyWith 模拟封装层的显式签名：直接声明 ...copier.Option 形参并透传。
func copyWith(dst, src any, opts ...copier.Option) error {
	return copier.Copy(dst, src, opts...)
}

// 编译期断言：所有 With* 工厂返回值类型均兼容 copier.Option。
var _ = []copier.Option{
	copier.WithMaxDepth(3),
	copier.WithIgnoreEmpty(),
	copier.WithCaseSensitive(),
	copier.WithMust(),
	copier.WithConverters(),
	copier.WithTagName("copier"),
	copier.WithSkipFields("X"),
	copier.WithValueConverter(nil),
	copier.WithNameMapping(nil),
	copier.WithNameFn(nil),
	copier.WithMethodMapping(),
}

type optionExportSrc struct {
	Name   string
	Secret string
}

type optionExportDst struct {
	Name   string
	Secret string
}

func TestOptionExportWithSkipFields(t *testing.T) {
	src := optionExportSrc{Name: "n", Secret: "s"}
	var dst optionExportDst

	err := copyWith(&dst, src, copier.WithSkipFields("Secret"))
	assert.NoError(t, err)
	assert.Equal(t, "n", dst.Name)
	assert.Equal(t, "", dst.Secret) // skip 字段经选项透传不复制
}

func TestOptionExportWithIgnoreEmpty(t *testing.T) {
	src := optionExportSrc{Name: "n"} // Secret 为零值
	dst := optionExportDst{Secret: "keep"}

	err := copyWith(&dst, src, copier.WithIgnoreEmpty())
	assert.NoError(t, err)
	assert.Equal(t, "keep", dst.Secret) // 零值字段跳过，保留初始值
}

// 方法映射选项同样可经 copier.Option 透传（外部包定义类型与方法）。
type optionExportSetterDst struct {
	Stored string
}

func (d *optionExportSetterDst) Name(v string) { d.Stored = v }

func TestOptionExportWithMethodMapping(t *testing.T) {
	src := optionExportSrc{Name: "x"}
	var dst optionExportSetterDst

	err := copyWith(&dst, src, copier.WithMethodMapping())
	assert.NoError(t, err)
	assert.Equal(t, "x", dst.Stored)
}

func TestOptionExportCaseSensitive(t *testing.T) {
	type dst struct {
		name string // 未导出，声明在前
		Name string
	}

	var d1, d2 dst
	// 默认（不敏感）：匹配第一个字段 name（未导出）→ 不可设置 → 跳过
	err := copyWith(&d1, optionExportSrc{Name: "x"})
	assert.NoError(t, err)
	assert.Equal(t, "", d1.Name)

	// 敏感：精确匹配 Name
	err = copyWith(&d2, optionExportSrc{Name: "x"}, copier.WithCaseSensitive())
	assert.NoError(t, err)
	assert.Equal(t, "x", d2.Name)
}
