package zormsql

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/milagre/zote/go/zelement/zsort"
)

// copyFields copies only the specified fields from src to dst.
func copyFields(dst, src reflect.Value, fields []string) {
	for _, f := range fields {
		srcField := src.Elem().FieldByName(f)
		if srcField.IsValid() {
			dstField := dst.Elem().FieldByName(f)
			if dstField.IsValid() && dstField.CanSet() {
				dstField.Set(srcField)
			}
		}
	}
}

func extractFields(fields []string, objPtr reflect.Value) []interface{} {
	values := make([]interface{}, 0, len(fields))
	for _, f := range fields {
		values = append(values, objPtr.Elem().FieldByName(f).Interface())
	}
	return values
}

func validateListOfPtr(listOfPtrs any) (reflect.Value, reflect.Type, error) {
	if listOfPtrs == nil {
		return reflect.Value{}, nil, fmt.Errorf("list of pointers required: nil provided")
	}

	targetList := reflect.ValueOf(listOfPtrs)
	if targetList.Type().Kind() != reflect.Slice {
		return reflect.Value{}, nil, fmt.Errorf("list of pointers required: non-list provided")
	}

	modelPtrType := targetList.Type().Elem()
	if modelPtrType.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, fmt.Errorf("list of pointers required: list of non-pointer types provided")
	}

	return targetList, modelPtrType, nil
}

func validatePtrToListOfPtr(ptrToListOfPtrs any) (reflect.Value, reflect.Type, error) {
	if ptrToListOfPtrs == nil {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: nil provided")
	}

	targetVal := reflect.ValueOf(ptrToListOfPtrs)
	if targetVal.Type().Kind() != reflect.Ptr {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: non-pointer provided")
	}

	if targetVal.Type().Elem().Kind() != reflect.Slice {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: non-list provided")
	}

	targetList := targetVal.Elem()

	modelPtrType := targetVal.Type().Elem().Elem()
	if modelPtrType.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: list of non-pointer types provided")
	}

	return targetList, modelPtrType, nil
}

// createNullableScanTarget creates a nullable scan target (sql.NullString, sql.NullInt64, etc.)
// for a given field type. This is used for relation columns that might be NULL.
func createNullableScanTarget(fieldType reflect.Type) interface{} {
	// Handle pointer types - get the underlying type
	elemType := fieldType
	if fieldType.Kind() == reflect.Ptr {
		elemType = fieldType.Elem()
	}

	switch elemType.Kind() {
	case reflect.String:
		return &sql.NullString{}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &sql.NullInt64{}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Use NullInt64 for unsigned integers (may lose precision for very large uint64)
		return &sql.NullInt64{}
	case reflect.Float32, reflect.Float64:
		return &sql.NullFloat64{}
	case reflect.Bool:
		return &sql.NullBool{}
	default:
		// For time.Time and other types, try to use the pointer type
		if elemType == reflect.TypeOf(time.Time{}) {
			return &sql.NullTime{}
		}
		// For unknown types, fall back to pointer (may fail on NULL, but that's the current behavior)
		return reflect.New(fieldType).Interface()
	}
}

// sortRelations sorts to-many relation slices in memory based on their Sort configuration.
func sortRelations(obj reflect.Value, s structure) error {
	for _, name := range s.relations {
		rel, ok := s.toManyRelations[name]
		if !ok {
			// to-one relation, check for nested relations
			if rel, ok = s.toOneRelations[name]; ok {
				field := obj.FieldByName(name)
				if !field.IsNil() {
					if err := sortRelations(field.Elem(), rel.structure); err != nil {
						return fmt.Errorf("sorting nested relation %s: %w", name, err)
					}
				}
			}
			continue
		}

		field := obj.FieldByName(name)
		if field.IsNil() || field.Len() == 0 {
			continue
		}

		// Sort if Sort is configured
		if len(rel.sort) > 0 {
			if err := zsort.ApplyToValue(field, rel.sort); err != nil {
				return fmt.Errorf("sorting relation %s: %w", name, err)
			}
		}

		// Recurse into nested relations
		for i := 0; i < field.Len(); i++ {
			elem := field.Index(i)
			if elem.Kind() == reflect.Ptr && !elem.IsNil() {
				if err := sortRelations(elem.Elem(), rel.structure); err != nil {
					return fmt.Errorf("sorting nested relation %s[%d]: %w", name, i, err)
				}
			}
		}
	}
	return nil
}
