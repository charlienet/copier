package copier

import "reflect"

// Clone 深拷贝 src 并返回同类型副本（仅同类型，T 由参数推导）。
// 语义：容器字段（slice/map/指针）深拷贝隔离，struct 值类型字段遵循值拷贝语义。
// 指针类型返回独立分配的新对象。出错时返回 error 与零值副本。
// 需 panic 语义时用 Copy(...).Must()：
//
//	var dst T
//	copier.Copy(src, &dst).Must()
//
// 跨类型转换（含 struct→map）请用 Copy：
//
//	var m map[string]any
//	err := copier.Copy(src, &m).Do()
func Clone[T any](src T) (T, error) {
	var dst T
	if err := copyInto(&dst, src); err != nil {
		var zero T
		return zero, err
	}
	return dst, nil
}

// copyInto 将 src 深拷贝到 *dst，内部复用 copier 内核（v0.3 起 Copy 已删除，
// 直接以 DefaultOptions 拷贝调用内核）。
// dst 为指针类型时（如 Clone[*Foo] 的 var dst *Foo 为 nil 指针），
// copier 的目标经 indirect 解到具体值，nil 指针解到底层为无效值会报
// ErrInvalidCopyDestination，故先分配整条 nil 指针链使目标可寻址。
func copyInto[T any](dst *T, src any) error {
	v := reflect.ValueOf(dst).Elem()
	for v.Kind() == reflect.Pointer && v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
		v = v.Elem()
	}
	opt := *DefaultOptions
	return copier(dst, src, &opt)
}
