package zreflect

import (
	"fmt"
	"reflect"
)

func TypeID(t reflect.Type) string {
	namePrefix := ""
	if t.Kind() == reflect.Ptr {
		namePrefix = "*"
		t = t.Elem()
	}

	result := fmt.Sprintf("%s.%s%s", t.PkgPath(), namePrefix, t.Name())

	return result
}

func MakeAddressableSliceOf(valueType reflect.Type, len int, cap int) reflect.Value {
	sliceType := reflect.SliceOf(valueType)

	result := reflect.New(sliceType).Elem()
	result.Set(reflect.MakeSlice(sliceType, len, cap))

	return result
}
