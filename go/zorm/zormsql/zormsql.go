package zormsql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zsql"
)

var _ zorm.Repository = &Repository{}
var _ Connection = zsql.NewConnection(nil, zsql.NewDriver("_"))

type Queryable interface {
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
}

type Transaction interface {
	Queryable
	Commit() error
	Rollback() error
}

type Connection interface {
	Queryable

	Driver() string
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type Source struct {
	name string
	conn Connection
}

func NewSource(name string, conn Connection) Source {
	return Source{
		name: name,
		conn: conn,
	}
}

func (s Source) Name() string {
	return s.name
}

type Repository struct {
	mappers map[reflect.Type]Mapper
}

type Mapper struct {
	Type       reflect.Type
	Source     Source
	Table      string
	PrimaryKey []string
	Columns    Columns
	Relations  Relations
}

type Struct interface {
	struct{}
}

func NewMapper[T any](source Source, table string, pk []string, cols Columns, rels Relations) (Mapper, error) {
	t := reflect.TypeOf(new(T)).Elem()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return Mapper{}, fmt.Errorf("mappers must have struct types")
	}

	return Mapper{
		Type:       t,
		Source:     source,
		Table:      table,
		PrimaryKey: pk,
		Columns:    cols,
		Relations:  rels,
	}, nil
}

func New(mappers []Mapper) Repository {
	maps := make(map[reflect.Type]Mapper)
	for _, m := range mappers {
		maps[m.Type] = m
	}
	return Repository{
		mappers: maps,
	}
}

func (r Repository) Find(ctx context.Context, models any, opts zorm.FindOptions) error {
	return nil
}

func (r Repository) Get(ctx context.Context, model any, opts zorm.GetOptions) error {
	return nil
}

type Relation struct {
	Src []string
	Dst []string
}

type Field struct {
	name     string
	noInsert bool
	noUpdate bool
}

type Columns map[string]Field
type Relations map[string]Relation

func F(name string) Field {
	return Field{name: name}
}

func (f Field) Name() string {
	return f.name
}

func (f Field) NoInsert() Field {
	f.noInsert = true
	return f
}

func (f Field) NoUpdate() Field {
	f.noUpdate = true
	return f
}

func C(names ...string) []string {
	return names
}
