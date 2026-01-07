package zsql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// ScanJSON providers a generic implementation for the sql.Scanner interface for
// structs representing columns in a sql database. Implement the sql.Scanner
// interface on your struct and simply call this function for the
// implementation.
//
// Example:
//
//	func (s *ExampleStruct) Scan(value interface{}) error {
//		return zsql.ScanJSON(s, value)
//	}
func ScanJSON[T any](ptr *T, value interface{}) error {
	if value == nil {
		*ptr = *new(T)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []uint8:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into %T", value, ptr)
	}

	return json.Unmarshal(bytes, ptr)
}

// ValueJSON provides a generic implementation for the driver.Valuer interface
// for structs representing columns in a sql database. Implement the
// driver.Valuer interface on your struct and simply call this function for the
// implementation.

// Example:
//
//	func (s *ExampleStruct) Value() (driver.Value, error) {
//		return zsql.ValueJSON(s)
//	}
func ValueJSON[T any](ptr *T) (driver.Value, error) {
	if ptr == nil {
		return nil, nil
	}
	return json.Marshal(ptr)
}
