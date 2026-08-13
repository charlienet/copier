package copier

// Copier 泛型链式拷贝构建器：将 src 深拷贝到 dst，支持链式配置选项。
// 通过 Copy / Clone 创建，以 Do() / Result() 执行。字段未导出，只能链式修改。
// S=源类型，D=目标类型，由 Copy 调用参数自动推导。
// 注意：本包声明的 Copy 遮蔽内置 copy 函数；包内无内置 copy 使用点，
// 新增代码请勿使用内置 copy（用 slices.Clone 等替代）。
type Copier[S, D any] struct {
	src  S
	dst  *D
	opts options
}

// Copy 创建链式拷贝构建器。dst 必须为非 nil 指针。
// 参数顺序为 src 在前、dst 在后（与 json.Unmarshal(data, &v) 惯例一致；
// v0.2 的 Copy(dst, src, opts...) 已删除，顺序已反转）。
// S/D 可为 any（动态类型场景，如框架转发）。
// 默认严格模式（v0.2+），转换失败时 Do() 返回 ErrConversionFailed。
//
//	var dst MyDst
//	err := copier.Copy(src, &dst).IgnoreEmpty().CaseSensitive().Do()
func Copy[S, D any](src S, dst *D) *Copier[S, D] {
	return &Copier[S, D]{src: src, dst: dst, opts: *DefaultOptions}
}

// Do 执行深拷贝，内部复用 copier 内核（plan 缓存兼容）。
// Do 不修改构建器状态，同一 Copier 实例可重复执行，每次应用同一组选项；
// 对已填充 dst 为覆盖语义（匹配字段覆盖、未匹配字段保留）。
// Clone 入口请以 Result() 终结。
func (c *Copier[S, D]) Do() error {
	return copier(c.dst, c.src, &c.opts)
}

// Result 执行深拷贝并返回分配结果：成功返回 *c.dst（D 值），失败返回零值 + err。
// 内部复用 copier 内核，错误语义与 Do() 完全一致（失败时返回 D 零值与 error，
// 与 v0.3 的 Clone 语义相同）。
// D 为指针类型时（如 Clone[*Foo] 的 D=*Foo），成功返回新分配的指针（非 nil）。
// 对 Copy 入口创建的 builder 调用 Result() 返回 *c.dst 的值
// （与 dst 共享引用字段底层，非深拷贝；无害）。
func (c *Copier[S, D]) Result() (D, error) {
	if err := copier(c.dst, c.src, &c.opts); err != nil {
		var zero D
		return zero, err
	}
	return *c.dst, nil
}

// With 一次性应用 Config 中的非零字段配置（零值字段保持当前配置不变）。
// nil cfg 为 no-op（返回 c）。可与链式方法混用；With 立即应用非零字段，
// 零值字段不构成"设置"（不覆盖）。对同一选项，后设置者生效。
func (c *Copier[S, D]) With(cfg *Config) *Copier[S, D] {
	if cfg == nil {
		return c
	}
	cfg.apply(&c.opts)
	return c
}

// Lenient 显式退出严格模式：转换失败恢复静默跳过/留零值语义。
func (c *Copier[S, D]) Lenient() *Copier[S, D] {
	c.opts.strict = false
	return c
}

// Strict 显式开启严格模式：转换失败报错而非静默留零/跳过（v0.2 起默认已开启）。
// 与 Lenient() 互逆：Lenient() 之后再 Strict() 可恢复严格（双向闭环）。
func (c *Copier[S, D]) Strict() *Copier[S, D] {
	c.opts.strict = true
	return c
}

// AllowPrecisionLoss 严格模式下豁免数值精度损失检查（float→int 截断/溢出、
// float64→float32 舍入、int→float 超精确范围）。
// 仅豁免精度类转换失败；字符串解析失败/类型不匹配等仍报错
// （与 Lenient() 不同，后者关闭整个严格模式）。
func (c *Copier[S, D]) AllowPrecisionLoss() *Copier[S, D] {
	c.opts.allowPrecisionLoss = true
	return c
}

// IgnoreEmpty 跳过零值字段。
func (c *Copier[S, D]) IgnoreEmpty() *Copier[S, D] {
	c.opts.ignoreEmpty = true
	return c
}

// CaseSensitive 大小写敏感字段匹配。
func (c *Copier[S, D]) CaseSensitive() *Copier[S, D] {
	c.opts.caseSensitive = true
	return c
}

// MustFields 只拷贝带 copier:"must" 标签的字段。
func (c *Copier[S, D]) MustFields() *Copier[S, D] {
	c.opts.must = true
	return c
}

// Converters 注册类型转换器。
func (c *Copier[S, D]) Converters(cv ...TypeConverter) *Copier[S, D] {
	c.opts.converters = cv
	return c
}

// NameMapping 设置字段名映射。
func (c *Copier[S, D]) NameMapping(m map[string]string) *Copier[S, D] {
	c.opts.fieldNameMapping = m
	return c
}

// NameFn 设置字段名转换函数。
func (c *Copier[S, D]) NameFn(fn func(string) string) *Copier[S, D] {
	c.opts.nameConverter = fn
	return c
}

// TagName 使用自定义标签名。
func (c *Copier[S, D]) TagName(name string) *Copier[S, D] {
	c.opts.tagName = name
	return c
}

// SkipFields 跳过指定字段。
func (c *Copier[S, D]) SkipFields(fields ...string) *Copier[S, D] {
	c.opts.skipFields = fields
	return c
}

// ValueConverter 设置字段值转换函数。
func (c *Copier[S, D]) ValueConverter(fn func(string, any) any) *Copier[S, D] {
	c.opts.valueConverter = fn
	return c
}

// MethodMapping 启用方法→字段映射。
func (c *Copier[S, D]) MethodMapping() *Copier[S, D] {
	c.opts.methodMapping = true
	return c
}

// MaxDepth 限制最大递归深度。
func (c *Copier[S, D]) MaxDepth(n int) *Copier[S, D] {
	c.opts.maxDepth = n
	return c
}

// NilSrcZero 将 nil 源视为零值目标。
func (c *Copier[S, D]) NilSrcZero() *Copier[S, D] {
	c.opts.nilSrcZero = true
	return c
}
