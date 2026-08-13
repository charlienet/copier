package copier

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameConvertFn(t *testing.T) {
	// camelCase → PascalCase（替代原 stringx.Camel2Pascal，保持零依赖）
	camel2Pascal := func(s string) string {
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}

	opt := testOpt(func(o *options) { o.nameConverter = camel2Pascal })
	r := opt.NameConvert("test")

	assert.Equal(t, "Test", r)
}

func TestNameMapping(t *testing.T) {
	opt := testOpt(func(o *options) {
		o.fieldNameMapping = map[string]string{
			"test": "Test1111111",
		}
	})

	r := opt.NameConvert("test")
	assert.Equal(t, "Test1111111", r)
}
