package sqlbuilder

import (
	"strings"
	"testing"
)

func TestBuildQueryQuotesReservedIdentifiers(t *testing.T) {
	query := &Query{
		Conditions: []*Condition{{
			ColName:  "group",
			Operator: "=",
			Value:    &Value{ValueType: ValueType_String, StringValue: "operators"},
		}},
		OrderConditions: []*OrderCondition{{Name: "group", Order: "desc"}},
		Page:            &PageQuery{PageSize: 10, PageIndex: 2},
		ReturnTotal:     true,
	}

	pageSQL, totalSQL := BuildQuery("id,group", "user", 100, query, nil)
	for _, fragment := range []string{"select `id`,`group`", "from `user`", "where `group` = ?", "order by `group` desc", "limit ? OFFSET ?"} {
		if !strings.Contains(pageSQL.Sql, fragment) {
			t.Fatalf("BuildQuery() SQL = %q, missing %q", pageSQL.Sql, fragment)
		}
	}
	if totalSQL == nil || !strings.Contains(totalSQL.Sql, "count(`id`)") || !strings.Contains(totalSQL.Sql, "from `user`") {
		t.Fatalf("BuildQuery() total SQL = %#v", totalSQL)
	}
	if len(pageSQL.Params) != 3 || pageSQL.Params[1] != int64(10) || pageSQL.Params[2] != int64(20) {
		t.Fatalf("BuildQuery() params = %#v", pageSQL.Params)
	}
}

func TestBuildQueryPreservesProjectionExpressions(t *testing.T) {
	pageSQL, _ := BuildQuery("count(*)", "user", 100, &Query{}, nil)
	if !strings.Contains(pageSQL.Sql, "select count(*) from `user`") {
		t.Fatalf("BuildQuery() SQL = %q", pageSQL.Sql)
	}
}

func TestBuildQueryQuotesJoinIdentifiers(t *testing.T) {
	pageSQL, _ := BuildQuery("id,group.name", "user", 100, &Query{}, &JoinCondition{
		TableName:        "group",
		TableColName:     "group_id",
		JoinTableColName: "group.id",
		TableAlisa:       "group",
	})
	for _, fragment := range []string{
		"select `t`.`id`,`t1`.`name`",
		"from `user`",
		"as t left join `group` as t1",
		"on `t`.`group_id`=`t1`.`id`",
	} {
		if !strings.Contains(pageSQL.Sql, fragment) {
			t.Fatalf("BuildQuery() SQL = %q, missing %q", pageSQL.Sql, fragment)
		}
	}
}

func TestBuildUpdateQuotesIdentifiersAndSkipsEmptyConditions(t *testing.T) {
	updateSQL, err := BuildUpdate(
		"user",
		42,
		map[string]bool{"group": true},
		[]*Field{{ColName: "group", FieldValue: &Value{ValueType: ValueType_String, StringValue: "operators"}}},
		[]*Condition{{ColName: "created_at", Operator: ">", Value: &Value{ValueType: ValueType_Time}}},
	)
	if err != nil {
		t.Fatalf("BuildUpdate() error = %v", err)
	}
	if updateSQL.Sql != "update `user` set `group`=? where `id`=?" {
		t.Fatalf("BuildUpdate() SQL = %q", updateSQL.Sql)
	}
	if len(updateSQL.Params) != 2 {
		t.Fatalf("BuildUpdate() params = %#v", updateSQL.Params)
	}
}

func TestBuildConditionSkipsEmptyInValues(t *testing.T) {
	query := &Query{Conditions: []*Condition{
		{ColName: "id", Operator: "IN", Value: &Value{ValueType: ValueType_IntArray}},
		{ColName: "group", Operator: "=", Value: &Value{ValueType: ValueType_String, StringValue: "operators"}},
	}}

	pageSQL, _ := BuildQuery("id", "user", 100, query, nil)
	if !strings.Contains(pageSQL.Sql, "where `group` = ?") || strings.Contains(pageSQL.Sql, "`id` in") {
		t.Fatalf("BuildQuery() SQL = %q", pageSQL.Sql)
	}
}

func TestAppendLimitUsesCanonicalSyntax(t *testing.T) {
	context := AppendLimit("select `id` from `user`", nil, 100, &PageQuery{PageSize: 10, PageIndex: 2})
	if context.Sql != "select `id` from `user` limit ? OFFSET ?" {
		t.Fatalf("AppendLimit() SQL = %q", context.Sql)
	}
	if len(context.Params) != 2 || context.Params[0] != int64(10) || context.Params[1] != int64(20) {
		t.Fatalf("AppendLimit() params = %#v", context.Params)
	}
}
