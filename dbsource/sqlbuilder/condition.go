package sqlbuilder

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/coffeehc/base/log"
	"github.com/lib/pq"
)

func NewCondtion(colName string, operator string, value *Value) *Condition {
	return &Condition{
		ColName:  colName,
		Operator: operator,
		Value:    value,
	}
}

func getConditionFieldValue(condition *Condition) interface{} {
	value := condition.GetValue()
	operator := condition.GetOperator()
	if value == nil {
		return nil
	}
	switch value.ValueType {
	case ValueTypeInt:
		return value.IntValue
	case ValueTypeDoubel:
		return value.DoubleValue
	case ValueTypeString:
		if strings.ToLower(operator) == "like" {
			return fmt.Sprintf("%%%s%%", value.StringValue)
		}
		return value.StringValue
	case ValueTypeBool:
		return value.BoolValue
	case ValueTypeTime:
		if value.IntValue == 0 {
			return nil
		}
		return time.Unix(0, value.IntValue)
	case ValueTypeIntArray:
		return value.IntValues
	case ValueTypeStringArray:
		return value.StringValues
	case ValueTypeDoubleArray:
		return value.DoubleValues
	case ValueTypeTimeArray:
		ts := make([]time.Time, 0, len(value.IntValues))
		for _, v := range value.IntValues {
			if v != 0 {
				ts = append(ts, time.Unix(0, v))
			}
		}
		return ts
	case ValueTypeStatus:
		if value.IntValue <= 0 {
			return nil
		}
		return value.IntValue
	case ValueTypeStatuses:
		return value.IntValues
	case ValueTypePgIntArray:
		return pq.Array(value.IntValues)
	case ValueTypePgFloatArray:
		return pq.Array(value.GetDoubleValues())
	case ValueTypePgStringArray:
		return pq.Array(value.GetStringValues())
	default:
		log.Error("未知的数据类型")
	}
	return nil
}

func buildCondition(sqlBuilder *strings.Builder, replace AlisaDefined, conditions []*Condition) []interface{} {
	params := make([]interface{}, 0)
	for _, condition := range conditions {
		if condition == nil || condition.ColName == "" || condition.Value == nil || condition.Operator == "" {
			continue
		}
		value := getConditionFieldValue(condition)
		if value == nil {
			continue
		}
		if len(params) > 0 {
			sqlBuilder.WriteString(" and ")
		}
		sqlBuilder.WriteString(replace.handle(condition.GetColName()))
		sqlBuilder.WriteString(" ")
		sqlBuilder.WriteString(condition.GetOperator())
		if condition.GetOperator() == "in" {
			sqlBuilder.WriteString(" ( ")
			v := reflect.ValueOf(value)
			for i := 0; i < v.Len(); i++ {
				if i > 0 {
					sqlBuilder.WriteString(" , ")
				}
				sqlBuilder.WriteString(" ? ")
				params = append(params, v.Index(i).Interface())
			}
			sqlBuilder.WriteString(" ) ")
		} else {
			sqlBuilder.WriteString(" ? ")
			params = append(params, value)
		}

	}
	return params
}
