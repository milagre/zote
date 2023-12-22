package zormsqlite3_test

import (
	"testing"

	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zsql"
	"github.com/stretchr/testify/assert"
)

func TestSimpleFind(t *testing.T) {
	assert.True(t, true)

	setup(t, func(c zsql.Connection, r zorm.Repository) {

	})
}
