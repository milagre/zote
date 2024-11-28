package zormsql

import (
	"fmt"

	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/milagre/zote/go/zsql"
)

type sortVisitor struct {
	driver            zsql.Driver
	table             table
	columnAliasPrefix string
	mapping           Mapping
}

func (v *sortVisitor) Visit(s zsort.Sort) (string, []interface{}, error) {
	ev := elemVisitor{
		driver:            v.driver,
		table:             v.table,
		columnAliasPrefix: v.columnAliasPrefix,
		mapping:           v.mapping,
	}

	result := ""

	elem, vals, err := ev.Visit(s.Element)
	if err != nil {
		return "", nil, fmt.Errorf("visiting sort element %v: %w", s.Element, err)
	}

	result += elem
	result += " "

	switch s.Direction {
	case zsort.Asc:
		result += "ASC"
	case zsort.Desc:
		result += "DESC"
	default:
		return "", nil, fmt.Errorf("invalid sort: %v", s.Direction)
	}

	return result, vals, nil
}
