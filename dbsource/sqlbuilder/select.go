package sqlbuilder

import (
	"fmt"
	"strings"
)

// BuildQuery creates one portable LIMIT/OFFSET query and its optional total-count query.
func BuildQuery(colNames, tableName string, maxPageSize int64, query *Query, joinCondition *JoinCondition) (pageSqlContext *SqlContext, totalSqlContext *SqlContext) {
	sqlBuilder := &strings.Builder{}
	params := make([]interface{}, 0)
	joinSql := ""
	if joinCondition == nil && query.GetJoin() != nil {
		joinCondition = query.GetJoin()
	}
	replace := aliasDefinition{
		openJoin: joinCondition != nil,
	}
	if replace.openJoin {
		replace.alisa = "t."
		replace.tableAlisa = joinCondition.TableAlisa
		replace.joinAlisa = "t1."
		joinSql = fmt.Sprintf(" as t left join %s as t1 on %s=%s", quoteIdentifier(joinCondition.TableName), replace.handle(joinCondition.TableColName), replace.handle(joinCondition.JoinTableColName))
	}
	// 预处理查询字段
	_colNames := strings.Split(colNames, ",")
	for i, colName := range _colNames {
		if i > 0 {
			sqlBuilder.WriteString(",")
		}
		sqlBuilder.WriteString(replace.handleProjection(colName))
	}
	colNames = sqlBuilder.String()
	sqlBuilder.Reset()
	// pageIndexReset:=false
	// 构建查询条件
	params = append(params, buildCondition(sqlBuilder, replace, query.GetConditions())...)
	conditionSql := sqlBuilder.String()
	if len(params) > 0 {
		conditionSql = fmt.Sprintf("where %s", conditionSql)
	}
	// 构建排序
	sqlBuilder.Reset()
	if len(query.GetOrderConditions()) > 0 {
		sqlBuilder.WriteString("order by ")
		for i, orderCondition := range query.GetOrderConditions() {
			if i > 0 {
				sqlBuilder.WriteString(",")
			}
			sqlBuilder.WriteString(fmt.Sprintf("%s %s", replace.handle(orderCondition.GetName()), orderCondition.Order))
			// if pageIndexReset && query.GetPage() != nil {
			//   query.Page.PageIndex = 0
			// }
		}
	}
	orderSql := sqlBuilder.String()
	sqlBuilder.Reset()
	// 构建分页语句
	limitSql := ""
	pageParams := make([]interface{}, 0)
	page := query.GetPage()
	if page != nil {
		if page.GetPageSize() <= 0 {
			page.PageSize = 30
		}
		if page.GetPageSize() > maxPageSize {
			page.PageSize = maxPageSize
		}
		if page.GetPageIndex() < 0 {
			page.PageIndex = 0
		}
		limitSql = " limit ? OFFSET ?"
		pageParams = append(pageParams, page.GetPageSize(), page.GetPageIndex()*page.GetPageSize())
	}
	pageSqlContext = &SqlContext{
		Sql:    fmt.Sprintf("select %s from %s %s %s %s %s", colNames, quoteIdentifier(tableName), joinSql, conditionSql, orderSql, limitSql),
		Params: append(params, pageParams...),
	}
	if query.GetReturnTotal() {
		pk := "id"
		if query.Pk != "" {
			pk = query.Pk
		}
		totalSqlContext = &SqlContext{
			Sql:    fmt.Sprintf("select count(%s) as `count` from %s %s %s", replace.handle(pk), quoteIdentifier(tableName), joinSql, conditionSql),
			Params: params,
		}
	}
	return pageSqlContext, totalSqlContext
}

// AppendLimit adds canonical LIMIT/OFFSET placeholders to an existing query.
func AppendLimit(sql string, param []interface{}, maxPageSize int64, query *PageQuery) *SqlContext {
	if query.GetPageSize() < 0 || query.GetPageSize() > maxPageSize {
		query.PageSize = maxPageSize
	}
	if query.GetPageSize() > 0 {
		param = append(param, query.GetPageSize())
		param = append(param, query.GetPageIndex()*query.GetPageSize())
		sql = fmt.Sprintf("%s limit ? OFFSET ?", sql)
	}
	return &SqlContext{sql, param}
}
