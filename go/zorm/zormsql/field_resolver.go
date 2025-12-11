package zormsql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/milagre/zote/go/zreflect"
)

// isSimpleField returns true if the path contains no dots (i.e., it's a simple field, not a relation path)
func isSimpleField(path string) bool {
	return !strings.Contains(path, ".")
}

// relationStep represents one step in navigating a relation path
type relationStep struct {
	relationName    string
	relMapping      Relation
	relationMapping Mapping
	leftTable       table
	rightTable      table
	pathAlias       string
}

// navigateRelationPath navigates through a dot-delimited field path and calls the callback
// for each relation step. The callback receives information about each step and can return
// an error to stop navigation early.
func navigateRelationPath(cfg *Config, mapping Mapping, startTable table, path string, callback func(step relationStep) error) error {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid dot-delimited field path: %s", path)
	}

	currentMapping := mapping
	currentTable := startTable
	pathParts := []string{}

	for i := 0; i < len(parts)-1; i++ {
		relationName := parts[i]

		pathParts = append(pathParts, relationName)
		pathAlias := strings.Join(pathParts, "_")

		relMapping, ok := currentMapping.relationByField(relationName)
		if !ok {
			return fmt.Errorf("relation %s not found in mapping", relationName)
		}

		ptrType := reflect.TypeOf(currentMapping.PtrType)
		structField, ok := ptrType.Elem().FieldByName(relationName)
		if !ok {
			return fmt.Errorf("field %s not found in type %T", relationName, currentMapping.PtrType)
		}

		var relationType reflect.Type
		if structField.Type.Kind() == reflect.Ptr {
			relationType = structField.Type
		} else if structField.Type.Kind() == reflect.Slice {
			relationType = structField.Type.Elem()
		} else {
			return fmt.Errorf("invalid relation type for field %s", relationName)
		}

		relationMapping, ok := cfg.mappings[zreflect.TypeID(relationType)]
		if !ok {
			return fmt.Errorf("mapping not found for relation type %s", relationType)
		}

		rightTable := table{
			name:  relationMapping.Table,
			alias: pathAlias,
		}

		step := relationStep{
			relationName:    relationName,
			relMapping:      relMapping,
			relationMapping: relationMapping,
			leftTable:       currentTable,
			rightTable:      rightTable,
			pathAlias:       pathAlias,
		}

		if err := callback(step); err != nil {
			return err
		}

		// Move to next relation
		currentMapping = relationMapping
		currentTable = rightTable
	}

	return nil
}

// resolveDotDelimitedField navigates through mappings to resolve a dot-delimited field path
// and returns the final column for the field.
func resolveDotDelimitedField(cfg *Config, mapping Mapping, startTable table, path string) (column, error) {
	// Extract field name (everything after the last dot)
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return column{}, fmt.Errorf("invalid dot-delimited field path: %s", path)
	}
	fieldName := path[lastDot+1:]

	var finalTable table
	var finalMapping Mapping

	err := navigateRelationPath(cfg, mapping, startTable, path, func(step relationStep) error {
		finalTable = step.rightTable
		finalMapping = step.relationMapping
		return nil
	})
	if err != nil {
		return column{}, err
	}

	col, _, err := finalMapping.mapField(finalTable, "", fieldName)
	if err != nil {
		return column{}, fmt.Errorf("field %s not found in relation mapping: %w", fieldName, err)
	}

	return col, nil
}
