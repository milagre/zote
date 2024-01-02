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

func IsInt(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return true
	}
	return false
}

func IsString(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String:
		return true
	}
	return false
}
