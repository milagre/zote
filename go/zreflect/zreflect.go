package zreflect

import "reflect"

func TypeID(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}

func MakeAddressableSliceOf(valueType reflect.Type, len int, cap int) reflect.Value {
	sliceType := reflect.SliceOf(valueType)

	result := reflect.New(sliceType).Elem()
	result.Set(reflect.MakeSlice(sliceType, len, cap))

	return result
}
