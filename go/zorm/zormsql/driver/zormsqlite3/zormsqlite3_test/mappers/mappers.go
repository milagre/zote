package zormsqlite3_test_mappers

import (
	"fmt"

	s "github.com/milagre/zote/go/zorm/zormsql"

	structs "github.com/milagre/zote/go/zorm/zormsql/driver/zormsqlite3/zormsqlite3_test/structs"
)

var AccountMapper s.Mapper
var UserMapper s.Mapper

func Mappers(source s.Source) []s.Mapper {
	res := []s.Mapper{}

	mapper, err := s.NewMapper[structs.Account](
		source,
		"accounts",
		[]string{"ID"},
		s.Columns{
			"id":      s.F("ID").NoInsert().NoUpdate(),
			"company": s.F("Company"),
		},
		s.Relations{},
	)
	if err != nil {
		panic(fmt.Sprintf("Account mapper invalid: %+v", err))
	}
	res = append(res, mapper)

	mapper, err = s.NewMapper[structs.User](
		source,
		"users",
		[]string{"ID"},
		s.Columns{
			"id":   s.F("ID").NoInsert().NoUpdate(),
			"name": s.F("Name"),
		},
		s.Relations{
			"Account": s.Relation{Src: s.C("AccountID"), Dst: s.C("ID")},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("User mapper invalid: %+v", err))
	}
	res = append(res, mapper)

	return res
}
