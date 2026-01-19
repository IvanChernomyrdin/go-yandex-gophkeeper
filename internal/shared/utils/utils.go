// Утилитарные функции общего назначения
package utils

func Ptr[T any](v T) *T {
	return &v
}

func StrPtr(s string) *string {
	return &s
}
