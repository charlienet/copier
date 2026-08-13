package copier

// Clone 深拷贝 src 并返回同类型副本（仅同类型，T 由参数推导）。
// 语义：容器字段（slice/map/指针）深拷贝隔离，struct 值类型字段遵循值拷贝语义。
// 指针类型返回独立分配的新对象。
//
// 返回 builder（*Copier[T, T]），可链式追加选项后以终端方法执行：
//
//	got, err := copier.Clone(src).Result() // 失败返回零值 + error
//
// Result() 是唯一终端：失败时返回 D 零值与 error，由调用方显式处理
// （需要 fail-fast 时自行 if err != nil { panic(err) }）。
// 跨类型转换（含 struct→map）请用 Copy：
//
//	var m map[string]any
//	err := copier.Copy(src, &m).Do()
func Clone[T any](src T) *Copier[T, T] {
	var dst T
	return &Copier[T, T]{src: src, dst: &dst, opts: *DefaultOptions}
}
