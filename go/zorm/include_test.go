package zorm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zelem"
)

func TestInclude_Merge_Fields(t *testing.T) {
	tests := []struct {
		name  string
		left  Fields
		right Fields
		want  Fields
	}{
		{name: "empty loads all", left: Fields{}, right: Fields{"Name"}, want: nil},
		{name: "all on the right loads all", left: Fields{"Name"}, right: nil, want: nil},
		{name: "affirmative union", left: Fields{"Name"}, right: Fields{"Email", "Name"}, want: Fields{"Name", "Email"}},
		{name: "negated keeps shared exclusions", left: Fields{"-Name", "-Email"}, right: Fields{"-Email"}, want: Fields{"-Email"}},
		{name: "negated with no shared exclusions loads all", left: Fields{"-Name"}, right: Fields{"-Email"}, want: nil},
		{name: "requested field lifts exclusion", left: Fields{"-Name", "-Email"}, right: Fields{"Name"}, want: Fields{"-Email"}},
		{name: "exclusion does not remove requested field", left: Fields{"Name"}, right: Fields{"-Name", "-Email"}, want: Fields{"-Email"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Include{Fields: tt.left}.Merge(Include{Fields: tt.right})

			assert.Equal(t, tt.want, got.Fields)
		})
	}
}

func TestInclude_Merge_Relations(t *testing.T) {
	t.Run("union merges shared relations recursively", func(t *testing.T) {
		left := Include{Relations: Relations{
			"Owner": {Include: Include{Relations: Relations{"Address": {}}}},
			"Tags":  {},
		}}
		right := Include{Relations: Relations{
			"Owner":    {Include: Include{Relations: Relations{"Company": {}}}},
			"Comments": {},
		}}

		got := left.Merge(right)

		assert.Equal(t, Include{Relations: Relations{
			"Owner":    {Include: Include{Relations: Relations{"Address": {}, "Company": {}}}},
			"Tags":     {},
			"Comments": {},
		}}, got)
	})

	t.Run("does not modify either side", func(t *testing.T) {
		left := Include{Relations: Relations{"Owner": {Include: Include{Relations: Relations{"Address": {}}}}}}
		right := Include{Relations: Relations{"Owner": {Include: Include{Relations: Relations{"Company": {}}}}}}

		_ = left.Merge(right)

		assert.Equal(t, Relations{"Address": {}}, left.Relations["Owner"].Include.Relations)
		assert.Equal(t, Relations{"Company": {}}, right.Relations["Owner"].Include.Relations)
	})

	t.Run("where is the union of both filters", func(t *testing.T) {
		active := zelem.Eq(zelem.Field("Active"), zelem.Value(true))
		recent := zelem.Gt(zelem.Field("Age"), zelem.Value(1))

		got := Include{Relations: Relations{"Tags": {Where: active}}}.
			Merge(Include{Relations: Relations{"Tags": {Where: recent}}})

		assert.Equal(t, zclause.Clause(zelem.Or(active, recent)), got.Relations["Tags"].Where)
	})

	t.Run("an unfiltered side loads everything", func(t *testing.T) {
		active := zelem.Eq(zelem.Field("Active"), zelem.Value(true))

		got := Include{Relations: Relations{"Tags": {Where: active}}}.
			Merge(Include{Relations: Relations{"Tags": {}}})

		assert.Nil(t, got.Relations["Tags"].Where)
	})

	t.Run("receiver sort wins", func(t *testing.T) {
		byName := zelem.Sorts(zelem.Asc(zelem.Field("Name")))
		byAge := zelem.Sorts(zelem.Desc(zelem.Field("Age")))

		got := Include{Relations: Relations{"Tags": {Sort: byName}}}.
			Merge(Include{Relations: Relations{"Tags": {Sort: byAge}}})

		assert.Equal(t, byName, got.Relations["Tags"].Sort)

		got = Include{Relations: Relations{"Tags": {}}}.
			Merge(Include{Relations: Relations{"Tags": {Sort: byAge}}})

		assert.Equal(t, byAge, got.Relations["Tags"].Sort)
	})
}
