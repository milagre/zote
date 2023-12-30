package zreflect

import "reflect"

func TypeID(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}
