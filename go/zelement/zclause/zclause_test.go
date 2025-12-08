package zclause_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
)

type mockVisitor struct {
	returnErr error
}

func (m *mockVisitor) VisitEq(eq zclause.Eq) error        { return m.returnErr }
func (m *mockVisitor) VisitNeq(neq zclause.Neq) error     { return m.returnErr }
func (m *mockVisitor) VisitGt(gt zclause.Gt) error        { return m.returnErr }
func (m *mockVisitor) VisitGte(gte zclause.Gte) error     { return m.returnErr }
func (m *mockVisitor) VisitLt(lt zclause.Lt) error        { return m.returnErr }
func (m *mockVisitor) VisitLte(lte zclause.Lte) error     { return m.returnErr }
func (m *mockVisitor) VisitNot(not zclause.Not) error     { return m.returnErr }
func (m *mockVisitor) VisitAnd(and zclause.And) error     { return m.returnErr }
func (m *mockVisitor) VisitOr(or zclause.Or) error        { return m.returnErr }
func (m *mockVisitor) VisitIn(in zclause.In) error        { return m.returnErr }
func (m *mockVisitor) VisitTruthy(t zclause.Truthy) error { return m.returnErr }

func TestClauseAccept(t *testing.T) {
	expectedErr := errors.New("test error")
	v := &mockVisitor{returnErr: expectedErr}

	tests := []struct {
		name   string
		clause zclause.Clause
	}{
		{"Eq", zclause.Eq{Left: zelement.Value{Value: 1}, Right: zelement.Value{Value: 1}}},
		{"Neq", zclause.Neq{Left: zelement.Value{Value: 1}, Right: zelement.Value{Value: 2}}},
		{"Gt", zclause.Gt{Left: zelement.Value{Value: 2}, Right: zelement.Value{Value: 1}}},
		{"Gte", zclause.Gte{Left: zelement.Value{Value: 2}, Right: zelement.Value{Value: 2}}},
		{"Lt", zclause.Lt{Left: zelement.Value{Value: 1}, Right: zelement.Value{Value: 2}}},
		{"Lte", zclause.Lte{Left: zelement.Value{Value: 1}, Right: zelement.Value{Value: 1}}},
		{"Not", zclause.Not{Clause: zclause.Eq{}}},
		{"And", zclause.And{Clauses: []zclause.Clause{zclause.Eq{}, zclause.Neq{}}}},
		{"Or", zclause.Or{Clauses: []zclause.Clause{zclause.Eq{}, zclause.Neq{}}}},
		{"In", zclause.In{
			Left:  []zelement.Element{zelement.Value{Value: 1}},
			Right: [][]zelement.Element{{zelement.Value{Value: 1}}},
		}},
		{"Truthy", zclause.Truthy{Elem: zelement.Value{Value: true}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.clause.Accept(v)
			assert.Equal(t, expectedErr, err)
		})
	}
}
