package sqlbuilder

import (
	"fmt"
	"strings"
)

type AlisaDefined struct {
	tableAlisa string
	alisa      string
	joinAlisa  string
	openJoin   bool
}

func (impl AlisaDefined) handle(colName string) string {
	if !impl.openJoin {
		return colName
	}
	colName = strings.TrimSpace(colName)
	if strings.Index(colName, ".") > 0 {
		return strings.Replace(colName, impl.tableAlisa, impl.joinAlisa, -1)
	}
	return fmt.Sprintf("%s%s", impl.alisa, colName)
}
