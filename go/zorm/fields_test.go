package zorm

import (
	"reflect"
	"testing"
)

func TestFields_Add(t *testing.T) {
	tests := []struct {
		name      string
		fields    Fields
		addFields []string
		want      Fields
	}{
		{
			name:      "add to empty fields",
			fields:    Fields{},
			addFields: []string{"ID"},
			want:      Fields{"ID"},
		},
		{
			name:      "add existing field is no-op",
			fields:    Fields{"Name", "Email"},
			addFields: []string{"Name"},
			want:      Fields{"Name", "Email"},
		},
		{
			name:      "add new field to affirmative list",
			fields:    Fields{"Name"},
			addFields: []string{"Email"},
			want:      Fields{"Name", "Email"},
		},
		{
			name:      "add to negated list removes exclusion",
			fields:    Fields{"-ID", "-Created"},
			addFields: []string{"ID"},
			want:      Fields{"-Created"},
		},
		{
			name:      "add non-excluded field to negated list is no-op",
			fields:    Fields{"-Created"},
			addFields: []string{"ID"},
			want:      Fields{"-Created"},
		},
		{
			name:      "add multiple fields to negated list",
			fields:    Fields{"-ID", "-Created", "-Modified"},
			addFields: []string{"ID", "Modified"},
			want:      Fields{"-Created"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.Add(tt.addFields...)
			if !reflect.DeepEqual(tt.fields, tt.want) {
				t.Errorf("Fields.Add() = %v, want %v", tt.fields, tt.want)
			}
		})
	}
}

func TestFields_IsNegated(t *testing.T) {
	tests := []struct {
		name   string
		fields Fields
		want   bool
	}{
		{
			name:   "empty fields is not negated",
			fields: Fields{},
			want:   false,
		},
		{
			name:   "nil fields is not negated",
			fields: nil,
			want:   false,
		},
		{
			name:   "affirmative fields is not negated",
			fields: Fields{"Name", "Email"},
			want:   false,
		},
		{
			name:   "negated fields is negated",
			fields: Fields{"-Created", "-Modified"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.IsNegated()
			if got != tt.want {
				t.Errorf("Fields.IsNegated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFields_Resolve(t *testing.T) {
	allFields := []string{"ID", "Name", "Created", "Modified", "Email"}

	tests := []struct {
		name      string
		fields    Fields
		allFields []string
		want      []string
	}{
		{
			name:      "empty fields returns all fields",
			fields:    Fields{},
			allFields: allFields,
			want:      allFields,
		},
		{
			name:      "nil fields returns all fields",
			fields:    nil,
			allFields: allFields,
			want:      allFields,
		},
		{
			name:      "affirmative fields returns only those fields",
			fields:    Fields{"Name", "Email"},
			allFields: allFields,
			want:      []string{"Name", "Email"},
		},
		{
			name:      "negated fields excludes specified fields",
			fields:    Fields{"-Created", "-Modified"},
			allFields: allFields,
			want:      []string{"ID", "Name", "Email"},
		},
		{
			name:      "single negated field",
			fields:    Fields{"-Email"},
			allFields: allFields,
			want:      []string{"ID", "Name", "Created", "Modified"},
		},
		{
			name:      "negated field not in allFields is ignored",
			fields:    Fields{"-NotAField"},
			allFields: allFields,
			want:      allFields,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.Resolve(tt.allFields)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Fields.Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}
