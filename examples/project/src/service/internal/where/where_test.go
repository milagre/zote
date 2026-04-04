package where

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWhere_Empty(t *testing.T) {
	c, err := ParseWhere("  ")
	require.NoError(t, err)
	require.Nil(t, c)
}

func TestParseWhere_NameEq(t *testing.T) {
	c, err := ParseWhere(`name == "foo"`)
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestParseWhere_AndOr(t *testing.T) {
	c, err := ParseWhere(`name == "a" && name != "b"`)
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestParseWhere_InvalidField(t *testing.T) {
	_, err := ParseWhere(`unknown == "x"`)
	require.Error(t, err)
}

func TestParseWhere_Trailing(t *testing.T) {
	_, err := ParseWhere(`name == "x" +`)
	require.Error(t, err)
}
