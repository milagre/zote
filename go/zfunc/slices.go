package zfunc

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func MapE[T, V any](ts []T, fn func(T) (V, error)) ([]V, error) {
	result := make([]V, len(ts))
	for i, t := range ts {
		v, err := fn(t)
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

func MakeSlice[T any](val T, len int) []T {
	result := make([]T, 0, len)
	for i := 0; i < len; i++ {
		result = append(result, val)
	}
	return result
}

func Select[T any](list []T, cb func(T) bool) []T {
	result := make([]T, 0, len(list))
	for _, elem := range list {
		if cb(elem) {
			result = append(result, elem)
		}
	}
	return result
}
