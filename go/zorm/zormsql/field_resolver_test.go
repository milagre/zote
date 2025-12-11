package zormsql

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zreflect"
)

// Test types for navigation tests
type navTestUser struct {
	ID      string
	Name    string
	Address *navTestAddress
}

type navTestAddress struct {
	ID    string
	State string
	City  string
}

// Test types for resolve tests
type resolveAccount struct {
	ID    string
	Name  string
	Users []*resolveUser
}

type resolveUser struct {
	ID      string
	Name    string
	Address *resolveAddress
}

type resolveAddress struct {
	ID    string
	State string
	City  string
}

func TestIsSimpleField(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Simple field",
			path:     "Name",
			expected: true,
		},
		{
			name:     "Dot-delimited path",
			path:     "Users.Name",
			expected: false,
		},
		{
			name:     "Nested dot-delimited path",
			path:     "Users.Address.State",
			expected: false,
		},
		{
			name:     "Empty string",
			path:     "",
			expected: true,
		},
		{
			name:     "Field with underscore",
			path:     "User_Name",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSimpleField(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNavigateRelationPath(t *testing.T) {
	// Setup test mappings
	userMapping := Mapping{
		PtrType:    &navTestUser{},
		Table:      "users",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Field: "ID"},
			{Name: "name", Field: "Name"},
		},
		Relations: []Relation{
			{
				Table: "addresses",
				Columns: map[string]string{
					"id": "address_id",
				},
				Field: "Address",
			},
		},
	}

	addressMapping := Mapping{
		PtrType:    &navTestAddress{},
		Table:      "addresses",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Field: "ID"},
			{Name: "state", Field: "State"},
			{Name: "city", Field: "City"},
		},
		Relations: []Relation{},
	}

	cfg := &Config{
		mappings: map[string]Mapping{
			zreflect.TypeID(reflect.TypeOf(&navTestUser{})):    userMapping,
			zreflect.TypeID(reflect.TypeOf(&navTestAddress{})): addressMapping,
		},
	}

	startTable := table{
		name:  "users",
		alias: "target",
	}

	t.Run("Single level relation", func(t *testing.T) {
		var steps []relationStep
		err := navigateRelationPath(cfg, userMapping, startTable, "Address.State", func(step relationStep) error {
			steps = append(steps, step)
			return nil
		})

		require.NoError(t, err)
		require.Len(t, steps, 1)

		step := steps[0]
		assert.Equal(t, "Address", step.relationName)
		assert.Equal(t, "Address", step.pathAlias)
		assert.Equal(t, "target", step.leftTable.alias)
		assert.Equal(t, "Address", step.rightTable.alias)
		assert.Equal(t, "addresses", step.rightTable.name)
		assert.Equal(t, addressMapping.Table, step.relationMapping.Table)
	})

	t.Run("Invalid path - no dots", func(t *testing.T) {
		err := navigateRelationPath(cfg, userMapping, startTable, "Name", func(step relationStep) error {
			return nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid dot-delimited field path")
	})

	t.Run("Missing relation", func(t *testing.T) {
		err := navigateRelationPath(cfg, userMapping, startTable, "Missing.State", func(step relationStep) error {
			return nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "relation Missing not found in mapping")
	})

	t.Run("Missing mapping for relation type", func(t *testing.T) {
		type Unknown struct {
			ID string
		}

		type navTestUserWithUnknown struct {
			ID      string
			Name    string
			Address *navTestAddress
			Unknown *Unknown
		}

		userMappingWithUnknown := Mapping{
			PtrType:    &navTestUserWithUnknown{},
			Table:      "users",
			PrimaryKey: []string{"id"},
			Columns: []Column{
				{Name: "id", Field: "ID"},
			},
			Relations: []Relation{
				{
					Table: "unknowns",
					Columns: map[string]string{
						"id": "unknown_id",
					},
					Field: "Unknown",
				},
			},
		}

		cfgMissing := &Config{
			mappings: map[string]Mapping{
				zreflect.TypeID(reflect.TypeOf(&navTestUserWithUnknown{})): userMappingWithUnknown,
			},
		}

		err := navigateRelationPath(cfgMissing, userMappingWithUnknown, startTable, "Unknown.ID", func(step relationStep) error {
			return nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "mapping not found for relation type")
	})

	t.Run("Callback error stops navigation", func(t *testing.T) {
		callbackError := assert.AnError
		err := navigateRelationPath(cfg, userMapping, startTable, "Address.State", func(step relationStep) error {
			return callbackError
		})

		require.Error(t, err)
		assert.Equal(t, callbackError, err)
	})
}

func TestResolveDotDelimitedField(t *testing.T) {
	// Setup test mappings
	accountMapping := Mapping{
		PtrType:    &resolveAccount{},
		Table:      "accounts",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Field: "ID"},
			{Name: "name", Field: "Name"},
		},
		Relations: []Relation{
			{
				Table: "users",
				Columns: map[string]string{
					"id": "account_id",
				},
				Field: "Users",
			},
		},
	}

	userMapping := Mapping{
		PtrType:    &resolveUser{},
		Table:      "users",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Field: "ID"},
			{Name: "name", Field: "Name"},
		},
		Relations: []Relation{
			{
				Table: "addresses",
				Columns: map[string]string{
					"id": "address_id",
				},
				Field: "Address",
			},
		},
	}

	addressMapping := Mapping{
		PtrType:    &resolveAddress{},
		Table:      "addresses",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Field: "ID"},
			{Name: "state", Field: "State"},
			{Name: "city", Field: "City"},
		},
		Relations: []Relation{},
	}

	cfg := &Config{
		mappings: map[string]Mapping{
			zreflect.TypeID(reflect.TypeOf(&resolveAccount{})): accountMapping,
			zreflect.TypeID(reflect.TypeOf(&resolveUser{})):    userMapping,
			zreflect.TypeID(reflect.TypeOf(&resolveAddress{})): addressMapping,
		},
	}

	startTable := table{
		name:  "accounts",
		alias: "target",
	}

	t.Run("Single level relation", func(t *testing.T) {
		col, err := resolveDotDelimitedField(cfg, userMapping, startTable, "Address.State")

		require.NoError(t, err)
		assert.Equal(t, "state", col.name)
		assert.Equal(t, "Address", col.table.alias)
		assert.Equal(t, "addresses", col.table.name)
	})

	t.Run("Nested relation path", func(t *testing.T) {
		col, err := resolveDotDelimitedField(cfg, accountMapping, startTable, "Users.Address.State")

		require.NoError(t, err)
		assert.Equal(t, "state", col.name)
		assert.Equal(t, "Users_Address", col.table.alias)
		assert.Equal(t, "addresses", col.table.name)
	})

	t.Run("Invalid path - no dots", func(t *testing.T) {
		col, err := resolveDotDelimitedField(cfg, userMapping, startTable, "Name")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid dot-delimited field path")
		assert.Equal(t, column{}, col)
	})

	t.Run("Missing relation", func(t *testing.T) {
		col, err := resolveDotDelimitedField(cfg, userMapping, startTable, "Missing.Field")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "relation Missing not found in mapping")
		assert.Equal(t, column{}, col)
	})

	t.Run("Missing final field", func(t *testing.T) {
		col, err := resolveDotDelimitedField(cfg, userMapping, startTable, "Address.MissingField")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "field MissingField not found in relation mapping")
		assert.Equal(t, column{}, col)
	})

	t.Run("Primary key field", func(t *testing.T) {
		col, err := resolveDotDelimitedField(cfg, userMapping, startTable, "Address.ID")

		require.NoError(t, err)
		assert.Equal(t, "id", col.name)
		assert.Equal(t, "Address", col.table.alias)
	})
}
