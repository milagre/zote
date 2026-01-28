package zsort

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/milagre/zote/go/zelement"
)

type Direction int8

const (
	Asc  Direction = iota
	Desc Direction = iota
)

type Sort struct {
	Element   zelement.Element
	Direction Direction
}

type Sorts []Sort

// Apply sorts the slice in place based on the provided sorts.
// Elements can be structs or pointers to structs.
func Apply[T any](slice []T, sorts Sorts) error {
	if len(slice) < 2 || len(sorts) == 0 {
		return nil
	}

	sort.Slice(slice, func(i, j int) bool {
		return compareBySorts(reflect.ValueOf(slice[i]), reflect.ValueOf(slice[j]), sorts)
	})

	return nil
}

// ApplyToValue sorts a reflect.Value slice in place based on the provided sorts.
// This is useful when working with dynamically typed slices via reflection.
func ApplyToValue(slice reflect.Value, sorts Sorts) (err error) {
	recover := func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}
	defer recover()

	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("value is not a slice: %T", slice.Interface())
	}

	if slice.Len() < 2 || len(sorts) == 0 {
		return nil
	}

	sort.Slice(slice.Interface(), func(i, j int) bool {
		return compareBySorts(slice.Index(i), slice.Index(j), sorts)
	})

	return nil
}

func compareBySorts(a, b reflect.Value, sorts Sorts) bool {
	// Dereference pointers
	if a.Kind() == reflect.Ptr {
		a = a.Elem()
	}
	if b.Kind() == reflect.Ptr {
		b = b.Elem()
	}

	if a.Kind() != reflect.Struct {
		panic(fmt.Errorf("value is not a struct: %T", a.Interface()))
	}
	if b.Kind() != reflect.Struct {
		panic(fmt.Errorf("value is not a struct: %T", b.Interface()))
	}

	for _, s := range sorts {
		fieldName := extractFieldName(s.Element)
		if fieldName == "" {
			continue
		}

		aField := a.FieldByName(fieldName)
		bField := b.FieldByName(fieldName)

		cmp := compareValues(aField, bField)
		if cmp == 0 {
			continue
		}

		if s.Direction == Desc {
			return cmp > 0
		}
		return cmp < 0
	}
	return false
}

// extractFieldName extracts the field name from a sort element.
func extractFieldName(elem zelement.Element) string {
	f, ok := elem.(zelement.Field)
	if !ok {
		panic(fmt.Errorf("element is not a field: %v", elem))
	}

	return f.Name
}

// compareValues compares two reflect.Values and returns -1, 0, or 1.
func compareValues(a, b reflect.Value) int {
	if !a.IsValid() || !b.IsValid() {
		if !a.IsValid() && !b.IsValid() {
			return 0
		}
		if !a.IsValid() {
			return -1
		}
		return 1
	}

	switch a.Kind() {
	case reflect.String:
		as, bs := a.String(), b.String()
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ai, bi := a.Int(), b.Int()
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ai, bi := a.Uint(), b.Uint()
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0

	case reflect.Float32, reflect.Float64:
		af, bf := a.Float(), b.Float()
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0

	case reflect.Bool:
		ab, bb := a.Bool(), b.Bool()
		if ab == bb {
			return 0
		}
		if !ab && bb {
			return -1
		}
		return 1

	default:
		// For other types (like time.Time), try to use String representation
		as := fmt.Sprintf("%v", a.Interface())
		bs := fmt.Sprintf("%v", b.Interface())
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	}
}
