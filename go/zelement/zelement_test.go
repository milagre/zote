package zelement_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement"
)

type mockVisitor struct {
	visitValueCalled  bool
	visitFieldCalled  bool
	visitMethodCalled bool
	returnErr         error
}

func (m *mockVisitor) VisitValue(e zelement.Value) error {
	m.visitValueCalled = true
	return m.returnErr
}

func (m *mockVisitor) VisitField(e zelement.Field) error {
	m.visitFieldCalled = true
	return m.returnErr
}

func (m *mockVisitor) VisitMethod(e zelement.Method) error {
	m.visitMethodCalled = true
	return m.returnErr
}

func TestValue_Accept(t *testing.T) {
	v := &mockVisitor{}
	e := zelement.Value{Value: "test"}

	err := e.Accept(v)
	assert.NoError(t, err)
	assert.True(t, v.visitValueCalled, "VisitValue was not called")

	// Test error propagation
	expectedErr := errors.New("test error")
	v = &mockVisitor{returnErr: expectedErr}
	err = e.Accept(v)
	assert.Equal(t, expectedErr, err)
}

func TestField_Accept(t *testing.T) {
	v := &mockVisitor{}
	e := zelement.Field{Name: "test"}

	err := e.Accept(v)
	assert.NoError(t, err)
	assert.True(t, v.visitFieldCalled, "VisitField was not called")
}

func TestMethod_Accept(t *testing.T) {
	v := &mockVisitor{}
	e := zelement.Method{
		Name: "test",
		Params: []zelement.Element{
			zelement.Value{Value: "param"},
		},
	}

	err := e.Accept(v)
	assert.NoError(t, err)
	assert.True(t, v.visitMethodCalled, "VisitMethod was not called")
}
