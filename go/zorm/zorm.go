package zorm

import (
	"context"
	"fmt"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
)

var ErrNotFound = fmt.Errorf("not found")

type Repository interface {
	Find(ctx context.Context, ptrToListOfPtrs any, opts FindOptions) error
	Get(ctx context.Context, listOfPtrs any, opts GetOptions) error
}

type UpsertOptions struct {
	Include   Include
	Relations map[string]Include
}

type InsertOptions struct {
	Include   Include
	Relations map[string]Include
}

type DeleteOptions struct {
	Include   Include
	Relations map[string]Include
}

type FindOptions struct {
	Include Include
	Sort    zsort.Sort
	Where   zclause.Clause
	Offset  int
}

type GetOptions struct {
	Include Include
}

type Include struct {
	Fields    Fields
	Relations Relations
	Where     zclause.Clause
	Sort      []zsort.Sort
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

type Relations map[string]Include

func Get[T any](ctx context.Context, repo Repository, list []*T, opts GetOptions) error {
	return repo.Get(ctx, list, opts)
}

func Find[T any](ctx context.Context, repo Repository, list *[]*T, opts FindOptions) error {
	return repo.Find(ctx, list, opts)
}

/*

func Upsert[T any](ctx context.Context, list []T, opts UpsertOptions) error {
	return upsert(ctx, source, list, opts)
}

func Insert[T any](ctx context.Context, list []T, opts InsertOptions) error {
	return insert(ctx, source, list, opts)
}

func Delete[T any](ctx context.Context, list []T, opts DeleteOptions) error {
	return delete(ctx, source, list, opts)
}

func DeleteWhere[T any](ctx context.Context, list []T, clause zclause.Clause, opts DeleteOptions) (int, error) {
	return deleteWhere(ctx, source, list, clause, opts)
}

*/
