package zorm

import (
	"context"
	"fmt"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
)

var ErrNotFound = fmt.Errorf("not found")
var ErrConflict = fmt.Errorf("conflict")

type Queryer interface {
	Find(ctx context.Context, ptrToListOfPtrs any, opts FindOptions) error
	Get(ctx context.Context, listOfPtrs any, opts GetOptions) error
	Put(ctx context.Context, listOfPtrs any, opts PutOptions) error
	Delete(ctx context.Context, listOfPtrs any, opts DeleteOptions) error
}

type Beginner interface {
	Begin(ctx context.Context) (Transaction, error)
}

type Transaction interface {
	Queryer

	Commit() error
	Rollback() error
}

type Repository interface {
	Beginner
	Queryer
}

type PutOptions struct {
	Include    Include
	Relations  map[string]Include
	GetOptions GetOptions
}

type DeleteOptions struct {
	Include    Include
	Relations  map[string]Include
	GetOptions GetOptions
}

type FindOptions struct {
	Include Include
	Sort    []zsort.Sort
	Where   zclause.Clause
	Offset  int
}

type GetOptions struct {
	Include Include
}

type Include struct {
	Fields    Fields
	Relations Relations
	Sort      []zsort.Sort
}

type Relations map[string]Relation

type Relation struct {
	Include Include
	Where   zclause.Clause
}

type Fields []string

func (f *Fields) Add(fields ...string) {
	for _, newField := range fields {
		found := false
		for _, currField := range *f {
			if currField == newField {
				found = true
				break
			}
		}
		if !found {
			*f = append(*f, newField)
		}
	}
}

func Get[T any](ctx context.Context, repo Queryer, list []*T, opts GetOptions) error {
	return repo.Get(ctx, list, opts)
}

func Find[T any](ctx context.Context, repo Queryer, list *[]*T, opts FindOptions) error {
	return repo.Find(ctx, list, opts)
}

func Put[T any](ctx context.Context, repo Queryer, list []*T, opts PutOptions) error {
	return repo.Put(ctx, list, opts)
}

func Delete[T any](ctx context.Context, repo Queryer, list []*T, opts DeleteOptions) error {
	return repo.Delete(ctx, list, opts)
}

/*

func DeleteWhere[T any](ctx context.Context, list []T, clause zclause.Clause, opts DeleteOptions) (int, error) {
	return deleteWhere(ctx, source, list, clause, opts)
}

*/
