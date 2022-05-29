package sqlite3_test

import (
	"testing"

	zoteorm "github.com/milagre/zote/go/orm"
	zotesql "github.com/milagre/zote/go/sql"
	"github.com/stretchr/testify/assert"
)

func TestSimpleFind(t *testing.T) {
	assert.True(t, true)

	setup(t, func(c *zotesql.Connection, r zoteorm.Repository) {

	})
}
