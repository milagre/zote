package zfunc

func IndexBy[T any, U comparable](list []T, cb func(T) U) map[U]T {
	result := map[U]T{}

	for _, elem := range list {
		result[cb(elem)] = elem
	}

	return result
}

func GroupBy[T any, U comparable](list []T, cb func(T) U) map[U][]T {
	result := map[U][]T{}

	for _, elem := range list {
		result[cb(elem)] = append(result[cb(elem)], elem)
	}

	return result
}
