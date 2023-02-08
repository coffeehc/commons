package webfacade

import (
	"fmt"
	"github.com/coffeehc/commons/dbsource/sqlbuilder"
	"github.com/coffeehc/commons/sequences"
	"github.com/coffeehc/commons/utils"
	"github.com/gofiber/fiber/v2"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

func ParseQuery(c *fiber.Ctx, fieldMap map[string]FieldDefined, initCondition []*sqlbuilder.Condition) (*sqlbuilder.Query, error) {
	conditions, err := ParseCondition(c, fieldMap, initCondition)
	if err != nil {
		return nil, err
	}
	return &sqlbuilder.Query{
		Page:            ParsePageQuert(c),
		ReturnTotal:     GetBoolFromContext(c, "return_total"),
		Conditions:      conditions,
		OrderConditions: ParseOrderConditions(c, fieldMap),
	}, nil
}

type FieldDefined struct {
	VType         sqlbuilder.ValueType
	RealFieldName string
	Operator      string
	Convert       func(c *fiber.Ctx, key string) (*sqlbuilder.Value, error)
}

func ParsePageQuert(c *fiber.Ctx) *sqlbuilder.PageQuery {
	return &sqlbuilder.PageQuery{
		PageIndex: GetInt64FromContext(c, "page_index"),
		PageSize:  GetInt64FromContext(c, "page_size"),
	}
}

//type queryParams struct {
//	Sort []string `query:"sort"`
//}

func ParseOrderConditions(c *fiber.Ctx, fieldMap map[string]FieldDefined) []*sqlbuilder.OrderCondition {
	//keys := c.QueryArray("sort")
	keys := strings.Split(c.Query("sort"), ",")
	conditions := make([]*sqlbuilder.OrderCondition, 0, len(keys)+1)
	startId := GetInt64FromContext(c, "start_id")
	defined, ok := fieldMap["start_id"]
	if ok {
		if startId == 0 && defined.Operator == "<" {
			conditions = append(conditions, &sqlbuilder.OrderCondition{
				Name:  defined.RealFieldName,
				Order: "desc",
			})
		}
		if startId != 0 && defined.Operator != "<" {
			order := &sqlbuilder.OrderCondition{
				Name: defined.RealFieldName,
			}
			if defined.Operator == ">" {
				order.Order = "asc"
			} else {
				order.Order = "desc"
			}
			conditions = append(conditions, order)
		}
	}
	for _, key := range keys {
		defined, ok := fieldMap[key]
		if !ok {
			continue
		}
		v := c.Query(fmt.Sprintf("sort_%s", key))
		if v == "" || v != "asc" || v != "desc" {
			continue
		}
		conditions = append(conditions, &sqlbuilder.OrderCondition{
			Name:  defined.RealFieldName,
			Order: v,
		})
	}
	return conditions
}

func ParseCondition(c *fiber.Ctx, fieldMap map[string]FieldDefined, initCondition []*sqlbuilder.Condition) ([]*sqlbuilder.Condition, error) {
	//keys := c.QueryArray("field")
	keys := strings.Split(c.Query("field"), ",")
	if initCondition == nil {
		initCondition = []*sqlbuilder.Condition{}
	}
	conditions := make([]*sqlbuilder.Condition, 0, len(keys)+len(initCondition))
	conditions = append(conditions, initCondition...)
	for k, defined := range fieldMap {
		v := c.Query(k)
		if v == "" || v == "null" || v == "undefined" {
			continue
		}
		// c.QueryArray()
		param := &sqlbuilder.Value{
			ValueType: defined.VType,
		}
		var condition = &sqlbuilder.Condition{
			ColName:  defined.RealFieldName,
			Operator: defined.Operator,
			Value:    param,
		}
		if defined.Convert != nil {
			v, err := defined.Convert(c, k)
			if err != nil {
				log.Error("转换参数失败", zap.String("key", k), zap.Error(err))
				return nil, errors.MessageError("参数错误")
			}
			if v != nil {
				condition.Value = v
				conditions = append(conditions, condition)
			}
			continue
		}
		var err error = nil
		switch defined.VType {
		case sqlbuilder.ValueType_Int:
			param.IntValue, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				log.Error("Int64值解析失败", zap.String("value", v), zap.String("key", k))
				return nil, errors.MessageError("参数错误")
			}
			break
		case sqlbuilder.ValueType_IntArray, sqlbuilder.ValueType_Statuses:
			vs := strings.Split(c.Query(k), ",")
			//vs := c.QueryArray(k)
			ints := make([]int64, 0, len(vs))
			for _, _v := range vs {
				// if defined.Convert != nil {
				//   _v, err = defined.Convert(_v)
				//   if err != nil {
				//     log.Error("转换参数失败", errors.ConverError(err).GetFieldsWithCause()...)
				//     return nil, errors.MessageError("参数错误")
				//   }
				// }
				_vs := strings.Split(_v, ",")
				for _, v := range _vs {
					i, _ := strconv.ParseInt(v, 10, 64)
					if i > 0 || v != "" {
						ints = append(ints, i)
					}
				}

			}
			// log.Debug("获取", zap.Any("intss", ints))
			if len(ints) == 0 {
				condition = nil
				break
			}
			param.IntValues = ints
			break
		case sqlbuilder.ValueType_Doubel:
			param.DoubleValue, err = strconv.ParseFloat(v, 64)
			if err != nil {
				log.Error("Float64值解析失败", zap.String("value", v), zap.String("key", k))
				return nil, errors.MessageError("参数错误")
			}
			break
		case sqlbuilder.ValueType_Bool:
			param.BoolValue = v == "true" || v == "1"
		case sqlbuilder.ValueType_Time:
			t, err := time.Parse(utils.Format_TIME_YYYY_MM_DD_hh_mm_ss, v)
			if err != nil {
				log.Error("解析时间失败", zap.String("value", v), zap.String("key", k))
				return nil, errors.MessageError("参数错误")
			}
			param.IntValue = t.UnixNano()
		case sqlbuilder.ValueType_String:
			param.StringValue = v
		case sqlbuilder.ValueType_Status:
			param.IntValue, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				log.Error("状态值解析失败", zap.String("value", v), zap.String("key", k))
				return nil, errors.MessageError("参数错误")
			}
			if param.IntValue <= 0 {
				continue
			}
			break
		default:
			log.Warn("未知测数据类型", zap.Any("valueType", defined.VType), zap.String("key", k))
			return nil, errors.MessageError("参数错误")
		}
		if condition != nil {
			conditions = append(conditions, condition)
		}
	}
	return conditions, nil
}

func GetInt64FromContext(c *fiber.Ctx, key string) int64 {
	i, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return i
}

func GetBoolFromContext(c *fiber.Ctx, key string) bool {
	v := c.Query(key)
	return v == "true" || v == "1"
}

func GetTimeFormQuery(c *fiber.Ctx, key, layout string) (bool, time.Time, error) {
	timeStr := c.Query(key)
	if timeStr == "" {
		return false, time.Time{}, nil
	}
	t, err := time.Parse(layout, timeStr)
	if err != nil {
		return false, time.Time{}, errors.MessageError("时间解析失败")
	}
	return true, t, nil
}

// b6211fdd280145
func Int64IdDecodeConvert(value string) (string, error) {
	if len(value) > 15 {
		return value, nil
	}
	id := utils.Int64IdDecode(value)
	t := sequences.ParseSequenceToTime(id)
	if id == 0 || math.Abs(float64(time.Now().Sub(t))) > float64(time.Hour*24*365*3) {
		id, _ = strconv.ParseInt(value, 10, 64)
	}
	return strconv.FormatInt(id, 10), nil
}
