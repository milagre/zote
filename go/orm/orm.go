package orm

import (
	"context"

	"github.com/milagre/zote/go/element/clause"
	"github.com/milagre/zote/go/element/elem"
	"github.com/milagre/zote/go/element/sort"
)

var _ elem.Element
var _ sort.Sort

type Model interface {
}

type RepositoryStore struct{}

func (s RepositoryStore) Register(model Model, source Source, mapper Mapper) {

}

type Source interface {
	Name() string
}

func Repo[T Model](source Source) (Repository, error) {
	return nil, nil
}

type Repository interface {
	Find(ctx context.Context, models any, opts FindOptions) error
	Get(ctx context.Context, model any, opts GetOptions) error
}

type Mapper interface{}

type FindOptions struct {
	Where   clause.Clause
	Include Include
	Sort    sort.Sort
}

type GetOptions struct {
	Where   clause.Clause
	Include Include
}

type Include struct {
	Fields    []string
	Relations map[string]Include
	Where     clause.Clause
	Sort      sort.Sort
}
