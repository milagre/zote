package zsort_test

import (
	"testing"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Person struct {
	Name string
	Age  int
}

func TestApply(t *testing.T) {
	t.Run("sorts by string field ascending", func(t *testing.T) {
		people := []Person{
			{Name: "Charlie", Age: 30},
			{Name: "Alice", Age: 25},
			{Name: "Bob", Age: 35},
		}

		err := zsort.Apply(people, zsort.Sorts{
			{Element: zelement.Field{Name: "Name"}, Direction: zsort.Asc},
		})

		require.NoError(t, err)
		assert.Equal(t, "Alice", people[0].Name)
		assert.Equal(t, "Bob", people[1].Name)
		assert.Equal(t, "Charlie", people[2].Name)
	})

	t.Run("sorts by string field descending", func(t *testing.T) {
		people := []Person{
			{Name: "Alice", Age: 25},
			{Name: "Charlie", Age: 30},
			{Name: "Bob", Age: 35},
		}

		err := zsort.Apply(people, zsort.Sorts{
			{Element: zelement.Field{Name: "Name"}, Direction: zsort.Desc},
		})

		require.NoError(t, err)
		assert.Equal(t, "Charlie", people[0].Name)
		assert.Equal(t, "Bob", people[1].Name)
		assert.Equal(t, "Alice", people[2].Name)
	})

	t.Run("sorts by int field ascending", func(t *testing.T) {
		people := []Person{
			{Name: "Charlie", Age: 30},
			{Name: "Alice", Age: 25},
			{Name: "Bob", Age: 35},
		}

		err := zsort.Apply(people, zsort.Sorts{
			{Element: zelement.Field{Name: "Age"}, Direction: zsort.Asc},
		})

		require.NoError(t, err)
		assert.Equal(t, 25, people[0].Age)
		assert.Equal(t, 30, people[1].Age)
		assert.Equal(t, 35, people[2].Age)
	})

	t.Run("sorts pointers to structs", func(t *testing.T) {
		people := []*Person{
			{Name: "Charlie", Age: 30},
			{Name: "Alice", Age: 25},
			{Name: "Bob", Age: 35},
		}

		err := zsort.Apply(people, zsort.Sorts{
			{Element: zelement.Field{Name: "Name"}, Direction: zsort.Asc},
		})

		require.NoError(t, err)
		assert.Equal(t, "Alice", people[0].Name)
		assert.Equal(t, "Bob", people[1].Name)
		assert.Equal(t, "Charlie", people[2].Name)
	})

	t.Run("sorts by multiple fields", func(t *testing.T) {
		people := []Person{
			{Name: "Alice", Age: 30},
			{Name: "Alice", Age: 25},
			{Name: "Bob", Age: 35},
		}

		err := zsort.Apply(people, zsort.Sorts{
			{Element: zelement.Field{Name: "Name"}, Direction: zsort.Asc},
			{Element: zelement.Field{Name: "Age"}, Direction: zsort.Asc},
		})

		require.NoError(t, err)
		assert.Equal(t, "Alice", people[0].Name)
		assert.Equal(t, 25, people[0].Age)
		assert.Equal(t, "Alice", people[1].Name)
		assert.Equal(t, 30, people[1].Age)
		assert.Equal(t, "Bob", people[2].Name)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		var people []Person

		err := zsort.Apply(people, zsort.Sorts{
			{Element: zelement.Field{Name: "Name"}, Direction: zsort.Asc},
		})

		require.NoError(t, err)
		assert.Empty(t, people)
	})

	t.Run("handles single element slice", func(t *testing.T) {
		people := []Person{{Name: "Alice", Age: 25}}

		err := zsort.Apply(people, zsort.Sorts{
			{Element: zelement.Field{Name: "Name"}, Direction: zsort.Asc},
		})

		require.NoError(t, err)
		assert.Len(t, people, 1)
		assert.Equal(t, "Alice", people[0].Name)
	})

	t.Run("handles empty sorts", func(t *testing.T) {
		people := []Person{
			{Name: "Charlie", Age: 30},
			{Name: "Alice", Age: 25},
		}

		err := zsort.Apply(people, zsort.Sorts{})

		require.NoError(t, err)
		// Order unchanged
		assert.Equal(t, "Charlie", people[0].Name)
		assert.Equal(t, "Alice", people[1].Name)
	})
}
