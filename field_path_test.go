package copier

// 错误信息带字段路径：strict 模式下转换失败的错误包含字段名（或 map key），
// 便于定位失败字段；errors.Is(err, ErrConversionFailed) 仍成立（%w 链上哨兵在里层）。

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fpSrc struct {
	Title string
}

type fpDst struct {
	Title int
}

func TestFieldPathStructToStruct(t *testing.T) {
	src := fpSrc{Title: "abc"}
	var dst fpDst

	err := Copy(src, &dst).Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))
	assert.Contains(t, err.Error(), "Title:") // 字段路径前缀
}

func TestFieldPathLegacyPath(t *testing.T) {
	// 非 eligible 选项强制走旧路径，同样带字段名
	src := fpSrc{Title: "abc"}
	var dst fpDst

	err := Copy(src, &dst).SkipFields("__none__").Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))
	assert.Contains(t, err.Error(), "Title:")
}

func TestFieldPathNested(t *testing.T) {
	// 嵌套 struct：外层字段名 + 内层字段名形成路径
	type outerSrc struct {
		Inner fpSrc
	}
	type outerDst struct {
		Inner fpDst
	}

	src := outerSrc{Inner: fpSrc{Title: "abc"}}
	var dst outerDst

	err := Copy(src, &dst).Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))
	msg := err.Error()
	assert.Contains(t, msg, "Inner:")
	assert.Contains(t, msg, "Title:")
}

func TestFieldPathMapToStruct(t *testing.T) {
	// map→struct：错误含 map key 名
	src := map[string]any{"Title": "abc"}
	var dst fpDst

	err := Copy(src, &dst).Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))
	assert.Contains(t, err.Error(), "Title:")
}

// 错误链格式验证：`Title: copier: value conversion failed: cannot parse "abc" as int`
// （字段名 → 哨兵 → 内层详情，%w 链上哨兵在里层，errors.Is 可匹配）。
func TestFieldPathFormat(t *testing.T) {
	src := fpSrc{Title: "abc"}
	var dst fpDst

	err := Copy(src, &dst).Do()
	assert.True(t, errors.Is(err, ErrConversionFailed))
	msg := err.Error()
	assert.True(t, strings.HasPrefix(msg, "Title:"))
	assert.Contains(t, msg, "copier: value conversion failed")
	assert.Contains(t, msg, "cannot parse")
}
