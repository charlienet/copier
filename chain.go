package copier

// Copier 泛型链式拷贝构建器：将 src 深拷贝到 dst，支持链式配置选项。
// 通过 Copy 创建，以 Do() / Must() 执行。字段未导出，只能链式修改。
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
// Do 不修改构建器状态，同一 Copier 实例可重复调用。
func (c *Copier[S, D]) Do() error {
	return copier(c.dst, c.src, &c.opts)
}

// Must 执行深拷贝，出错时 panic(err)。用于"确定不会失败"的填充已有场景。
func (c *Copier[S, D]) Must() {
	if err := c.Do(); err != nil {
		panic(err)
	}
}

// Lenient 显式退出严格模式：转换失败恢复静默跳过/留零值语义。
func (c *Copier[S, D]) Lenient() *Copier[S, D] {
	c.opts.strict = false
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
// 命名为 MustFields 避免与泛型 Must* 变体语义混淆。
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
