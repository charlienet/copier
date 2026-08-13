package copier

import "errors"

var (
	ErrInvalidCopyDestination = errors.New("copier: destination must be valid and addressable")
	ErrInvalidCopyFrom        = errors.New("copier: source must be non-nil")
	ErrMapKeyNotMatch         = errors.New("copier: map key is not a string")
	ErrNotSupported           = errors.New("copier: type combination not supported")
	ErrMaxDepthExceeded       = errors.New("copier: max copy depth exceeded")
	ErrMethodReturnError      = errors.New("copier: method returned error")
	ErrConversionFailed       = errors.New("copier: value conversion failed")
)

// FieldPathError 结构化字段路径错误：字段级拷贝失败时的包装错误。
// Error() 返回 "字段名: 底层错误"，与既有 fmt.Errorf("%s: %w") 包装输出逐字节一致
// （零消息回归）；Unwrap() 返回底层错误，errors.Is / errors.As 链不受影响。
// 嵌套 struct 路径经 Unwrap 链逐层收集，外层字段先出现、内层后出现
// （如错误串 "Inner: N: copier: ..."）。
// Field 为该层包装使用的字段名（map 路径为 NameConvert 后的名字）。
type FieldPathError struct {
	Field string
	Err   error
}

func (e *FieldPathError) Error() string {
	return e.Field + ": " + e.Err.Error()
}

func (e *FieldPathError) Unwrap() error {
	return e.Err
}
