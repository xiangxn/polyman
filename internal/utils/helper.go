package utils

func Filter[T any](src []T, fn func(T) bool) []T {
	dst := make([]T, 0, len(src))
	for _, v := range src {
		if fn(v) {
			dst = append(dst, v)
		}
	}
	return dst
}
