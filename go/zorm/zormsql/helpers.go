package zormsql

import (
	"fmt"
	"reflect"
)

func extractFields(fields []string, val reflect.Value) []interface{} {
	values := make([]interface{}, 0, len(fields))
	for _, f := range fields {
		values = append(values, val.Elem().FieldByName(f).Interface())
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
