// Package zorm provides an ORM framework with a repository interface.
//
// zorm defines interfaces for CRUD operations (Find, Get, Put, Delete) with
// relation loading. The main SQL implementation is in the zormsql subpackage.
//
// # Basic Usage
//
// Operations use generic helper functions that work with any Repository or
// Transaction:
//
//	// Find multiple records
//	var users []*User
//	zorm.Find(ctx, repo, &users, zorm.FindOptions{
//		Where: zclause.Eq("active", true),
//		Sort:  []zsort.Sort{{Field: "Created", Desc: true}},
//	})
//
//	// Get by primary key
//	user := &User{ID: 123}
//	zorm.Get(ctx, repo, []*User{user}, zorm.GetOptions{})
//
//	// Insert or update (Put with empty ID inserts, with ID updates)
//	zorm.Put(ctx, repo, []*User{user}, zorm.PutOptions{})
//
//	// Delete
//	zorm.Delete(ctx, repo, []*User{user}, zorm.DeleteOptions{})
//
// # Transactions
//
// Repositories provide transaction support:
//
//	tx, _ := repo.Begin(ctx)
//	zorm.Put(ctx, tx, []*User{{Name: "Bob"}}, zorm.PutOptions{})
//	tx.Commit() // or tx.Rollback()
//
// See zormsql for the SQL-based implementation with mapping configuration.
package zorm

import (
	"context"
	"fmt"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
)

var (
	ErrNotFound = fmt.Errorf("not found")
	ErrConflict = fmt.Errorf("conflict")
)

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
}

type Relations map[string]Relation

type Relation struct {
	Include Include
	Where   zclause.Clause
	Sort    []zsort.Sort
}

type Fields []string

func (f *Fields) Add(fields ...string) {
	// If the list is negated, adding a field means removing its exclusion
	if f.IsNegated() {
		for _, newField := range fields {
			negatedField := "-" + newField
			// Remove the negated version of this field if present
			for i, currField := range *f {
				if currField == negatedField {
					*f = append((*f)[:i], (*f)[i+1:]...)
					break
				}
			}
		}
		return
	}

	// For affirmative fields, add if not already present
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

// IsNegated returns true if the Fields list contains negated fields (fields starting with "-").
// An empty list is not considered negated.
func (f Fields) IsNegated() bool {
	return len(f) > 0 && len(f[0]) > 0 && f[0][0] == '-'
}

// Resolve returns the effective list of fields based on the Fields configuration:
//   - If Fields is empty, returns allFields (all fields)
//   - If Fields contains negated fields (starting with "-"), returns allFields minus the negated fields
//   - Otherwise, returns the Fields list as-is
//
// Negated and affirmative fields are mutually exclusive - mixing them is undefined behavior.
func (f Fields) Resolve(allFields []string) []string {
	if len(f) == 0 {
		return allFields
	}

	if !f.IsNegated() {
		return f
	}

	// Build a set of fields to exclude
	excluded := make(map[string]bool, len(f))
	for _, field := range f {
		if len(field) > 0 && field[0] == '-' {
			excluded[field[1:]] = true
		}
	}

	// Return all fields except the excluded ones
	result := make([]string, 0, len(allFields))
	for _, field := range allFields {
		if !excluded[field] {
			result = append(result, field)
		}
	}
	return result
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
