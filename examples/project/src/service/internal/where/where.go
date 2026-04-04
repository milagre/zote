package where

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zclause"
)

// ParseWhere parses a Go-like boolean expression for GET /items filtering.
// Allowed field slugs: name, created, modified. Empty input returns (nil, nil).
func ParseWhere(s string) (zclause.Clause, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	p := &prs{s: s, i: 0}
	n, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.i < len(p.s) {
		return nil, fmt.Errorf("unexpected trailing input near %q", p.s[p.i:])
	}
	return nodeToClause(n)
}

type prs struct {
	s string
	i int
}

func (p *prs) skipSpace() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\t' || p.s[p.i] == '\n' || p.s[p.i] == '\r') {
		p.i++
	}
}

func (p *prs) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		if !strings.HasPrefix(p.s[p.i:], "||") {
			return left, nil
		}
		p.i += 2
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = binary{op: "||", left: left, right: right}
	}
}

func (p *prs) parseAnd() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		if !strings.HasPrefix(p.s[p.i:], "&&") {
			return left, nil
		}
		p.i += 2
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binary{op: "&&", left: left, right: right}
	}
}

func (p *prs) parseUnary() (node, error) {
	p.skipSpace()
	if p.i < len(p.s) && p.s[p.i] == '!' && (p.i+1 >= len(p.s) || p.s[p.i+1] != '=') {
		p.i++
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unary{child: inner}, nil
	}
	return p.parseCompare()
}

func (p *prs) parseCompare() (node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	op, ok := p.scanRelOp()
	if !ok {
		return nil, fmt.Errorf("expected comparison operator")
	}
	right, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	return compare{op: op, left: left, right: right}, nil
}

func (p *prs) scanRelOp() (string, bool) {
	if strings.HasPrefix(p.s[p.i:], "==") {
		p.i += 2
		return "==", true
	}
	if strings.HasPrefix(p.s[p.i:], "!=") {
		p.i += 2
		return "!=", true
	}
	if strings.HasPrefix(p.s[p.i:], "<=") {
		p.i += 2
		return "<=", true
	}
	if strings.HasPrefix(p.s[p.i:], ">=") {
		p.i += 2
		return ">=", true
	}
	if p.i < len(p.s) {
		switch p.s[p.i] {
		case '<':
			p.i++
			return "<", true
		case '>':
			p.i++
			return ">", true
		}
	}
	return "", false
}

func (p *prs) parsePrimary() (operand, error) {
	p.skipSpace()
	if p.i >= len(p.s) {
		return nil, fmt.Errorf("unexpected end of input")
	}
	if p.s[p.i] == '(' {
		p.i++
		n, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if p.i >= len(p.s) || p.s[p.i] != ')' {
			return nil, fmt.Errorf("expected )")
		}
		p.i++
		return parenOp{inner: n}, nil
	}
	if p.s[p.i] == '"' {
		s, err := p.readString()
		if err != nil {
			return nil, err
		}
		return literalOp{kind: "string", str: s}, nil
	}
	if p.s[p.i] == '-' || p.s[p.i] == '+' || unicode.IsDigit(rune(p.s[p.i])) {
		n, err := p.readNumber()
		if err != nil {
			return nil, err
		}
		return literalOp{kind: "number", num: n}, nil
	}
	if unicode.IsLetter(rune(p.s[p.i])) || p.s[p.i] == '_' {
		id := p.readIdent()
		switch id {
		case "true", "false", "null":
			return literalOp{kind: id}, nil
		default:
			return fieldOp{slug: id}, nil
		}
	}
	return nil, fmt.Errorf("unexpected character %q", p.s[p.i])
}

func (p *prs) readString() (string, error) {
	p.i++ // "
	var b strings.Builder
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c == '"' {
			p.i++
			return b.String(), nil
		}
		if c == '\\' && p.i+1 < len(p.s) {
			p.i++
			switch p.s[p.i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				b.WriteByte(p.s[p.i])
			}
			p.i++
			continue
		}
		b.WriteByte(c)
		p.i++
	}
	return "", fmt.Errorf("unclosed string")
}

func (p *prs) readNumber() (float64, error) {
	start := p.i
	if p.s[p.i] == '-' || p.s[p.i] == '+' {
		p.i++
	}
	for p.i < len(p.s) && (unicode.IsDigit(rune(p.s[p.i])) || p.s[p.i] == '.') {
		p.i++
	}
	sub := p.s[start:p.i]
	f, err := strconv.ParseFloat(sub, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", sub)
	}
	return f, nil
}

func (p *prs) readIdent() string {
	start := p.i
	for p.i < len(p.s) {
		c := p.s[p.i]
		if unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '_' {
			p.i++
			continue
		}
		break
	}
	return p.s[start:p.i]
}

// --- AST ---

type node interface {
	nodeMarker()
}

type binary struct {
	op          string
	left, right node
}

func (binary) nodeMarker() {}

type unary struct {
	child node
}

func (unary) nodeMarker() {}

type compare struct {
	op         string
	left, right operand
}

func (compare) nodeMarker() {}

type parenOp struct {
	inner node
}

func (parenOp) nodeMarker()   {}
func (parenOp) operandMarker() {}

type operand interface {
	node
	operandMarker()
}

type fieldOp struct {
	slug string
}

func (fieldOp) nodeMarker()   {}
func (fieldOp) operandMarker() {}

type literalOp struct {
	kind string
	str  string
	num  float64
}

func (literalOp) nodeMarker()   {}
func (literalOp) operandMarker() {}

func nodeToClause(n node) (zclause.Clause, error) {
	switch v := n.(type) {
	case binary:
		lc, err := nodeToClause(v.left)
		if err != nil {
			return nil, err
		}
		rc, err := nodeToClause(v.right)
		if err != nil {
			return nil, err
		}
		switch v.op {
		case "&&":
			return zelem.And(lc, rc), nil
		case "||":
			return zelem.Or(lc, rc), nil
		default:
			return nil, fmt.Errorf("unknown op %s", v.op)
		}
	case unary:
		inner, err := nodeToClause(v.child)
		if err != nil {
			return nil, err
		}
		return zelem.Not(inner), nil
	case compare:
		return compareToClause(v)
	case parenOp:
		return nodeToClause(v.inner)
	default:
		return nil, fmt.Errorf("invalid expression")
	}
}

func fieldPath(slug string) (string, error) {
	switch slug {
	case "name":
		return "Items.Name", nil
	case "created":
		return "Items.Created", nil
	case "modified":
		return "Items.Modified", nil
	default:
		return "", fmt.Errorf("unknown field %q", slug)
	}
}

func compareToClause(c compare) (zclause.Clause, error) {
	if err := validateCompareOperands(c); err != nil {
		return nil, err
	}

	var fp string
	var value zelement.Element

	switch l := c.left.(type) {
	case fieldOp:
		var err error
		fp, err = fieldPath(l.slug)
		if err != nil {
			return nil, err
		}
		rl, ok := c.right.(literalOp)
		if !ok {
			return nil, fmt.Errorf("right side must be literal")
		}
		el, err := literalToElem(rl, l.slug)
		if err != nil {
			return nil, err
		}
		value = el
	case literalOp:
		fo, ok := c.right.(fieldOp)
		if !ok {
			return nil, fmt.Errorf("comparison must include one field")
		}
		var err error
		fp, err = fieldPath(fo.slug)
		if err != nil {
			return nil, err
		}
		el, err := literalToElem(l, fo.slug)
		if err != nil {
			return nil, err
		}
		value = el
	default:
		return nil, fmt.Errorf("comparison must include one field slug")
	}

	fieldEl := zelem.Field(fp)
	switch c.op {
	case "==":
		return zelem.Eq(fieldEl, value), nil
	case "!=":
		return zelem.Neq(fieldEl, value), nil
	case "<":
		return zelem.Lt(fieldEl, value), nil
	case "<=":
		return zelem.Lte(fieldEl, value), nil
	case ">":
		return zelem.Gt(fieldEl, value), nil
	case ">=":
		return zelem.Gte(fieldEl, value), nil
	default:
		return nil, fmt.Errorf("unknown op %q", c.op)
	}
}

func validateCompareOperands(c compare) error {
	switch l := c.left.(type) {
	case fieldOp:
		if _, err := fieldPath(l.slug); err != nil {
			return err
		}
		return checkLiteralForField(l.slug, c.right)
	case literalOp:
		fo, ok := c.right.(fieldOp)
		if !ok {
			return fmt.Errorf("comparison must include one field slug")
		}
		if _, err := fieldPath(fo.slug); err != nil {
			return err
		}
		return checkLiteralForField(fo.slug, c.left)
	default:
		return fmt.Errorf("comparison must include one field slug")
	}
}

func checkLiteralForField(slug string, o operand) error {
	lo, ok := o.(literalOp)
	if !ok {
		return fmt.Errorf("expected literal")
	}
	switch slug {
	case "name":
		switch lo.kind {
		case "string", "null":
			return nil
		default:
			return fmt.Errorf("name: use quoted string or null")
		}
	case "created", "modified":
		switch lo.kind {
		case "null":
			return nil
		case "string":
			if _, err := time.Parse(time.RFC3339, lo.str); err != nil {
				return fmt.Errorf("created/modified: RFC3339 time: %w", err)
			}
			return nil
		default:
			return fmt.Errorf("created/modified: quoted RFC3339 or null")
		}
	default:
		return nil
	}
}

func literalToElem(o literalOp, fieldSlug string) (zelement.Element, error) {
	switch o.kind {
	case "string":
		if fieldSlug == "created" || fieldSlug == "modified" {
			t, err := time.Parse(time.RFC3339, o.str)
			if err != nil {
				return nil, err
			}
			return zelem.Value(t), nil
		}
		return zelem.Value(o.str), nil
	case "number":
		return zelem.Value(o.num), nil
	case "true":
		return zelem.Value(true), nil
	case "false":
		return zelem.Value(false), nil
	case "null":
		return zelem.Value(nil), nil
	default:
		return nil, fmt.Errorf("bad literal")
	}
}
