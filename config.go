package copier

// Config 一次性批量配置（配合 Copier.With 使用）：字段与 options 内部配置一一对应，
// 采用"非零字段覆盖"语义——cfg 中非零值字段才覆盖到对应配置，零值字段表示
// "不设置、保持默认/当前值"。可与链式方法（IgnoreEmpty() 等）混用叠加。
//
// 各字段类型的覆盖规则：
//   - 字符串（TagName）：非空串才覆盖
//   - 整数（MaxDepth）：非 0 才覆盖（0 表示不设置；默认不限制见 DefaultOptions 的 noDepthLimited）
//   - bool（IgnoreEmpty / CaseSensitive / MustFields / MethodMapping / Lenient / NilSrcZero）：true 才覆盖
//   - 切片 / map / func（Converters / NameMapping / NameFn / SkipFields / ValueConverter）：非 nil 才覆盖
//
// 注：Lenient=true 时退出严格模式（strict=false）；默认严格模式见 DefaultOptions。
type Config struct {
	TagName        string                // 自定义标签名（非空才生效）
	MaxDepth       int                   // 最大递归深度（非 0 才生效；0=不设置）
	IgnoreEmpty    bool                  // 跳过零值字段（true 才生效）
	CaseSensitive  bool                  // 大小写敏感字段匹配（true 才生效）
	MustFields     bool                  // 只拷贝带 must 标签的字段（true 才生效）
	Converters     []TypeConverter       // 类型转换器（非 nil 才生效）
	NameMapping    map[string]string     // 字段名映射（非 nil 才生效）
	NameFn         func(string) string   // 字段名转换函数（非 nil 才生效）
	SkipFields     []string              // 跳过的字段列表（非 nil 才生效）
	ValueConverter func(string, any) any // 字段值转换函数（非 nil 才生效）
	MethodMapping  bool                  // 启用方法→字段映射（true 才生效）
	Lenient        bool                  // true 时退出严格模式（默认严格，见 DefaultOptions）
	NilSrcZero     bool                  // nil 源视为零值目标（true 才生效）
}

// apply 将 Config 的非零字段覆盖到 options 对应字段（零值字段保持原值不变）。
// 具体覆盖规则见 Config 的 doc comment。nil 接收者为 no-op。
func (c *Config) apply(o *options) {
	if c == nil {
		return
	}

	if c.TagName != "" {
		o.tagName = c.TagName
	}
	if c.MaxDepth != 0 {
		o.maxDepth = c.MaxDepth
	}
	if c.IgnoreEmpty {
		o.ignoreEmpty = true
	}
	if c.CaseSensitive {
		o.caseSensitive = true
	}
	if c.MustFields {
		o.must = true
	}
	if c.Converters != nil {
		o.converters = c.Converters
	}
	if c.NameMapping != nil {
		o.fieldNameMapping = c.NameMapping
	}
	if c.NameFn != nil {
		o.nameConverter = c.NameFn
	}
	if c.SkipFields != nil {
		o.skipFields = c.SkipFields
	}
	if c.ValueConverter != nil {
		o.valueConverter = c.ValueConverter
	}
	if c.MethodMapping {
		o.methodMapping = true
	}
	if c.Lenient {
		o.strict = false
	}
	if c.NilSrcZero {
		o.nilSrcZero = true
	}
}
