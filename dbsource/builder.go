package dbsource

import (
	"database/sql"
	"fmt"
	"github.com/coffeehc/commons/utils"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

func StructConvert(target interface{}, useTag string) (map[string]interface{}, error) {
	if nil == target {
		return nil, nil
	}
	v, err := reflectValueOf(target)
	if err != nil {
		return nil, err
	}
	t := v.Type()
	result := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		keyName := getKey(t.Field(i), useTag)
		if "" == keyName || "-" == keyName {
			continue
		}
		fieldValue := v.Field(i).Interface()
		switch fieldValue.(type) {
		// case string:
		// 	// if fieldValue.(string) == "" {
		// 	// 	continue
		// 	// }
		// case *string:
		// 	if fieldValue.(*string) != nil || *fieldValue.(*string) == "" {
		// 		continue
		// 	}
		// case int, int8, int16, int32, int64:
		// 	if fieldValue.(int64) == 0 {
		// 		continue
		// 	}
		// case time.Time:
		// 	if !fieldValue.(time.Time).IsZero() {
		// 		continue
		// 	}
		case sql.NullString:
			if !fieldValue.(sql.NullString).Valid {
				continue
			}
		case sql.NullInt64:
			if !fieldValue.(sql.NullInt64).Valid {
				continue
			}
		case sql.NullInt32:
			if !fieldValue.(sql.NullInt32).Valid {
				continue
			}
		case sql.NullTime:
			if !fieldValue.(sql.NullTime).Valid {
				continue
			}
		case sql.NullBool:
			if !fieldValue.(sql.NullBool).Valid {
				continue
			}
		case sql.NullFloat64:
			if !fieldValue.(sql.NullFloat64).Valid {
				continue
			}
		default:
		}
		result[keyName] = fieldValue
	}
	return result, nil
}

func getKey(field reflect.StructField, useTag string) string {
	if field.Type.Kind() == reflect.Ptr {
		return ""
	}
	if "" == useTag {
		return field.Name
	}
	tag, ok := field.Tag.Lookup(useTag)
	if !ok {
		return ""
	}
	return resolveTagName(tag)
}

func resolveTagName(tag string) string {
	idx := strings.IndexByte(tag, ',')
	if -1 == idx {
		return tag
	}
	return tag[:idx]
}

type SegmentType string

const (
	SegmentTypeInstall            SegmentType = "install"
	SegmentTypeUpdate             SegmentType = "update"
	SegmentTypeInstallOnDuplicate SegmentType = "installOnDuplicate"
)

func BuildSqlSegment(params map[string]interface{}, tablename string, segmentType SegmentType, pgFormat bool) (string, []interface{}, error) {
	switch segmentType {
	case SegmentTypeInstall:
		return buildInstallSegment(params, tablename)
	case SegmentTypeUpdate:
		return buildUpdateSegment(params, tablename, pgFormat)
	case SegmentTypeInstallOnDuplicate:
		return buildInstallOnDuplicateSegment(params, tablename)
	}
	return "", nil, errors.MessageError("不识别的片段类型")
}

func buildInstallOnDuplicateSegment(params map[string]interface{}, tableName string) (string, []interface{}, error) {
	count := len(params)
	keys := make([]string, 0, count)
	sortKeys := make([]string, 0, count)
	updateKeys := make([]string, 0, count)
	args := make([]interface{}, 0, count*2)
	updateArgs := make([]interface{}, 0, count)
	for k := range params {
		sortKeys = append(sortKeys, k)
	}
	sort.Strings(sortKeys)
	for _, k := range sortKeys {
		keys = append(keys, fmt.Sprintf("`%s`", k))
		v := params[k]
		args = append(args, v)
		updateKeys = append(updateKeys, fmt.Sprintf("`%s`=?", k))
		updateArgs = append(updateArgs, v)
	}
	sql := fmt.Sprintf("insert into %s (%s) value (%s)  ON DUPLICATE KEY UPDATE %s",
		tableName,
		strings.Join(keys, ","),
		strings.TrimRight(strings.Repeat("?,", count), ","),
		strings.Join(updateKeys, ","))

	return sql, append(args, updateArgs...), nil
}

func buildInstallSegment(params map[string]interface{}, tableName string) (string, []interface{}, error) {
	count := len(params)
	keys := make([]string, 0, count)
	args := make([]interface{}, 0, count)
	sortKeys := make([]string, 0, count)
	for k := range params {
		sortKeys = append(sortKeys, k)
	}
	sort.Strings(sortKeys)
	for _, k := range sortKeys {
		keys = append(keys, fmt.Sprintf("`%s`", k))
		args = append(args, params[k])
	}
	sql := fmt.Sprintf("insert into %s (%s) values (%s)", tableName, strings.Join(keys, ","), strings.TrimRight(strings.Repeat("?,", count), ","))
	return sql, args, nil
}

func buildUpdateSegment(params map[string]interface{}, tableName string, pgFormat bool) (string, []interface{}, error) {
	count := len(params)
	keys := make([]string, 0, count)
	args := make([]interface{}, 0, count)
	sortKeys := make([]string, 0, count)
	for k := range params {
		sortKeys = append(sortKeys, k)
	}
	sort.Strings(sortKeys)
	for _, k := range sortKeys {
		if pgFormat {
			keys = append(keys, fmt.Sprintf("%s=?", k))
		} else {
			keys = append(keys, fmt.Sprintf("`%s`=?", k))
		}
		args = append(args, params[k])
	}
	sql := strings.Join(keys, ",")
	return fmt.Sprintf("update %s set %s", tableName, sql), args, nil
}

func TableToDTOConvert(table interface{}, dto interface{}) error {
	table_v, err := reflectValueOf(table)
	if err != nil {
		return err
	}
	dto_v, err := reflectValueOf(dto)
	if err != nil {
		return err
	}
	fieldCount := table_v.NumField()
	for i := 0; i < fieldCount; i++ {
		fieldName := table_v.Type().Field(i).Name
		convertTableFieldToDto(table_v.FieldByName(fieldName), dto_v.FieldByName(fieldName))
	}
	return nil
}

func DTOToTableConvert(dto interface{}, table interface{}) error {
	table_v, err := reflectValueOf(table)
	if err != nil {
		return err
	}
	dto_v, err := reflectValueOf(dto)
	if err != nil {
		return err
	}
	fieldCount := table_v.NumField()
	for i := 0; i < fieldCount; i++ {
		fieldName := table_v.Type().Field(i).Name
		convertDtoFieldToTable(table_v.FieldByName(fieldName), dto_v.FieldByName(fieldName), fieldName)
	}
	return nil
}

var (
	typeSQlNullString  = reflect.TypeOf(sql.NullString{})
	typeSQlNullInt64   = reflect.TypeOf(sql.NullInt64{})
	typeSQlNullInt32   = reflect.TypeOf(sql.NullInt32{})
	typeSQlNullTime    = reflect.TypeOf(sql.NullTime{})
	typeSQlNullBool    = reflect.TypeOf(sql.NullBool{})
	typeSQlNullFloat64 = reflect.TypeOf(sql.NullFloat64{})
	typeBytes          = reflect.TypeOf([]byte{})
)

func convertDtoFieldToTable(table_field, dto_f reflect.Value, fieldName string) {
	if !dto_f.IsValid() || !table_field.IsValid() {
		return
	}
	if table_field.Type().ConvertibleTo(dto_f.Type()) {
		table_field.Set(dto_f.Convert(table_field.Type()))
		return
	}
	if dto_f.Kind() == reflect.Ptr {
		dto_f = dto_f.Elem()
	}
	switch dto_f.Kind() {
	case reflect.String:
		if table_field.Type().ConvertibleTo(typeSQlNullString) {
			table_field.FieldByName("Valid").SetBool(dto_f.String() != "")
			table_field.FieldByName("String").SetString(dto_f.String())
		}
		return
	case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
		switch table_field.Type().String() {
		case typeSQlNullInt64.String(), typeSQlNullInt32.String():
			table_field.FieldByNameFunc(func(s string) bool {
				return s == "Int64" || s == "Int32"
			}).SetInt(dto_f.Int())
			table_field.FieldByName("Valid").SetBool(true)
		case typeSQlNullTime.String():
			table_field.FieldByName("Time").Set(reflect.ValueOf(time.Unix(0, dto_f.Int())))
			table_field.FieldByName("Valid").SetBool(dto_f.Int() != 0)
		default:
			log.Warn("无法处理的类型", zap.String("type", table_field.Type().String()), zap.String("fieldName", fieldName))
		}
		return
	case reflect.Bool:
		if table_field.Type().ConvertibleTo(typeSQlNullBool) {
			table_field.FieldByName("Valid").SetBool(true)
			table_field.FieldByName("Bool").SetBool(dto_f.Bool())
		}
		return
	case reflect.Float64, reflect.Float32:
		if table_field.Type().ConvertibleTo(typeSQlNullFloat64) {
			table_field.FieldByName("Valid").SetBool(true)
			table_field.FieldByName("Float64").SetFloat(dto_f.Float())
		}
		return
	case reflect.Struct:
		if table_field.Type().ConvertibleTo(typeSQlNullTime) {
			table_field.FieldByName("Valid").SetBool(true)
			switch dto_f.Type().String() {
			case "time.Time":
				table_field.Set(dto_f)
			default:
				log.Warn("无法转换time类型", zap.String("dtoFieldType", dto_f.Type().String()), zap.String("fieldName", fieldName))
			}
		} else {
			if table_field.Type().ConvertibleTo(typeSQlNullInt64) {
				value := dto_f.FieldByName("Id")
				if value.IsValid() {
					if value.Kind() == reflect.Int64 {
						table_field.FieldByName("Valid").SetBool(true)
						table_field.FieldByNameFunc(func(s string) bool {
							return s == "Int64" || s == "Int32"
						}).SetInt(value.Int())
					}
				} else {
					log.Warn("不能识别的结构体", zap.String("type", table_field.Type().String()), zap.String("PkgPath", table_field.Type().PkgPath()), zap.String("fieldName", fieldName))
				}
			}
		}
		return
	case reflect.Slice:
		if table_field.Type().ConvertibleTo(typeSQlNullString) {
			table_field.FieldByName("Valid").SetBool(true)
			if dto_f.Type().String() == "[]int64" {
				ids := &strings.Builder{}
				for i := 0; i < dto_f.Len(); i++ {
					if i > 0 {
						ids.WriteString(",")
					}
					ids.WriteString(strconv.FormatInt(dto_f.Index(i).Int(), 10))
				}
				table_field.FieldByName("String").SetString(ids.String())
			} else {
				table_field.FieldByName("String").SetString(string(dto_f.Bytes()))
			}
			return
		} else {
			log.Warn("table字段不是String类型", zap.String("type", table_field.Type().String()), zap.String("PkgPath", table_field.Type().PkgPath()), zap.String("fieldName", fieldName))
		}
	case reflect.Array:
		if dto_f.Type().String() == "[]int64" {
			if table_field.Type().ConvertibleTo(typeSQlNullString) {
				ids := &strings.Builder{}
				for i := 0; i < dto_f.Len(); i++ {
					if i > 0 {
						ids.WriteString(",")
					}
					ids.WriteString(strconv.FormatInt(dto_f.Index(i).Int(), 10))
				}
				table_field.FieldByName("Valid").SetBool(true)
				table_field.FieldByName("String").SetString(ids.String())
			}
		}
	default:
		log.DPanic("不能识别的类型", zap.Any("dto_Kind", dto_f.Kind()), zap.String("type", table_field.Type().String()), zap.String("PkgPath", table_field.Type().PkgPath()), zap.String("fieldName", fieldName))
	}
}

func convertTableFieldToDto(table_field, dto_f reflect.Value) {
	if !dto_f.IsValid() || !table_field.IsValid() {
		return
	}
	if dto_f.Type().ConvertibleTo(table_field.Type()) {
		dto_f.Set(table_field.Convert(dto_f.Type()))
		return
	}
	valid := table_field.FieldByName("Valid")
	if !valid.Bool() {
		return
	}
	switch table_field.Type() {
	case typeSQlNullString:
		switch dto_f.Type().String() {
		case "[]uint8":
			dto_f.SetBytes([]byte(table_field.FieldByName("String").String()))
		case "string":
			dto_f.SetString(table_field.FieldByName("String").String())
		case "[]int64":
			str := table_field.FieldByName("String").String()
			ss := strings.Split(str, ",")
			vs := make([]int64, 0, len(ss))
			for _, s := range ss {
				i, _ := strconv.ParseInt(s, 10, 64)
				if i != 0 {
					vs = append(vs, i)
				}
			}
			dto_f.Set(reflect.ValueOf(vs))
		default:
			log.Error("无法转换的的字段类型", zap.String("dto_f.type", dto_f.Type().String()))
		}
	case typeSQlNullBool:
		dto_f.SetBool(table_field.FieldByName("Bool").Bool())
	case typeSQlNullTime:
		unixNano := table_field.FieldByName("Time").MethodByName("UnixNano").Call(nil)[0].Int()
		switch dto_f.Type().String() {
		case "time.Time":
			t := time.Unix(0, unixNano)
			dto_f.Set(reflect.ValueOf(t))
		case "string":
			t := time.Unix(0, unixNano).Format(utils.Format_TIME_YYYY_MM_DD_hh_mm_ss_SSS)
			dto_f.SetString(t)
		case "int64", "int32", "int16", "int8":
			dto_f.SetInt(unixNano)
		}
	case typeSQlNullInt64:
		switch dto_f.Kind() {
		case reflect.Int64, reflect.Int32:
			dto_f.SetInt(table_field.FieldByName("Int64").Int())
		case reflect.Ptr:
			if dto_f.IsNil() {
				return
			}
			dto_f = dto_f.Elem()
			if dto_f.Kind() == reflect.Struct {
				value := dto_f.FieldByName("Id")
				if value.IsValid() {
					value.SetInt(table_field.FieldByName("Int64").Int())
				}
			}
		}
	case typeSQlNullInt32:
		dto_f.SetInt(table_field.FieldByName("Int32").Int())
	case typeSQlNullFloat64:
		dto_f.SetFloat(table_field.FieldByName("Float64").Float())
	}
}

func reflectValueOf(obj interface{}) (reflect.Value, error) {
	if obj == nil {
		return reflect.Value{}, errors.SystemError("对象为空")
	}
	v := reflect.ValueOf(obj)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return v, errors.SystemError("目标不是一个结构体，无法转化")
	}
	return v, nil
}

func ConverFromPgsql(sql string) string {
	return strings.ReplaceAll(sql, "`", "")
}
