package sqlbuilder

import (
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

func BuildUpdate(tableName string, id int64, limitFields map[string]bool, fields []*Field, conditions []*Condition) (*SqlContext, error) {
	if id == 0 {
		return nil, errors.MessageError("没有指定Id")
	}
	sqlBuilder := new(strings.Builder)
	sqlBuilder.WriteString("update ")
	sqlBuilder.WriteString(tableName)
	sqlBuilder.WriteString(" set ")
	params := make([]interface{}, 0)
	for _, field := range fields {
		colName := field.GetColName()
		limitField, ok := limitFields[colName]
		if !ok || !limitField {
			log.Debug("limitField", zap.String("colName", colName))
			continue
		}
		value := getUpdateFieldValue(field.FieldValue)
		if value == nil {
			continue
		}
		if len(params) > 0 {
			sqlBuilder.WriteString(",")
		}
		sqlBuilder.WriteString(colName)
		sqlBuilder.WriteString("=?")
		params = append(params, value)
	}
	sqlBuilder.WriteString(" where ")
	sqlBuilder.WriteString(" id=? ")
	params = append(params, id)
	if len(conditions) > 0 {
		sqlBuilder.WriteString(" and ")
		params = append(params, buildCondition(sqlBuilder, AlisaDefined{}, conditions)...)
	}

	return &SqlContext{
		Sql:    sqlBuilder.String(),
		Params: params,
	}, nil
}

func getUpdateFieldValue(value *Value) interface{} {
	if value == nil {
		return nil
	}
	switch value.ValueType {
	case ValueTypeInt:
		return value.IntValue
	case ValueTypeDoubel:
		return value.DoubleValue
	case ValueTypeString:
		if value.StringValue == "" {
			return nil
		}
		return value.StringValue
	case ValueTypeBool:
		return value.BoolValue
	case ValueTypeTime:
		if value.IntValue == 0 {
			return nil
		}
		return time.Unix(0, value.IntValue)
	case ValueTypeStatus:
		if value.IntValue <= 0 {
			return nil
		}
		return value.IntValue
	default:
		log.Error("未知的数据类型")
	}
	return nil
}
