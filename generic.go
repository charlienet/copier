package copier

import "reflect"

// Clone 深拷贝 src 并返回同类型副本。
// 语义与 Copy(&dst, src) 完全一致：容器字段（slice/map/指针）深拷贝隔离，
// struct 值类型字段遵循值拷贝语义。指针类型返回独立分配的新对象。
// 出错时返回 error 与零值副本。
func Clone[T any](src T) (T, error) {
	var dst T
	if err := copyInto(&dst, src); err != nil {
		var zero T
		return zero, err
	}
	return dst, nil
}

// MustClone 同 Clone，出错时 panic(err)。
func MustClone[T any](src T) T {
	dst, err := Clone(src)
	if err != nil {
		panic(err)
	}
	return dst
}

// Convert 将 src 复制为 D 类型的新值（跨类型转换，如 DTO→BO）。
// 字段按名称匹配（默认大小写不敏感），支持自动类型转换与容器深拷贝。
// 出错时返回 error，dst 为零值。
func Convert[S, D any](src S) (D, error) {
	var dst D
	if err := copyInto(&dst, src); err != nil {
		var zero D
		return zero, err
	}
	return dst, nil
}

// MustConvert 同 Convert，出错时 panic(err)。
func MustConvert[S, D any](src S) D {
	dst, err := Convert[S, D](src)
	if err != nil {
		panic(err)
	}
	return dst
}

// copyInto 将 src 深拷贝到 *dst，内部复用 Copy。
// dst 为指针类型时（如 Clone[*Foo] 的 var dst *Foo 为 nil 指针），
// Copy 的目标经 indirect 解到具体值，nil 指针解到底层为无效值会报
// ErrInvalidCopyDestination，故先分配整条 nil 指针链使目标可寻址。
func copyInto[T any](dst *T, src any) error {
	v := reflect.ValueOf(dst).Elem()
	for v.Kind() == reflect.Pointer && v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
		v = v.Elem()
	}
	return Copy(dst, src)
}
