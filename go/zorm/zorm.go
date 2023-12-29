package zorm

import (
	"context"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
)

var _ zelement.Element
var _ zsort.Sort

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
	Where   zclause.Clause
	Include Include
	Sort    zsort.Sort
}

type GetOptions struct {
	Where   zclause.Clause
	Include Include
}

type Include struct {
	Fields    []string
	Relations map[string]Include
	Where     zclause.Clause
	Sort      zsort.Sort
}
