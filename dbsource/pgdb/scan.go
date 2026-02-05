package pgdb

import (
	"errors"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
)

type ScanAPI struct{}

// ScanAll 扫描多行
func (p *ScanAPI) ScanAll(ts interface{}, rows pgx.Rows) error {
	if ts == nil || rows == nil {
		return errors.New("ScanAll: ts or rows is nil")
	}

	sliceVal := reflect.ValueOf(ts)
	if sliceVal.Kind() != reflect.Ptr || sliceVal.Elem().Kind() != reflect.Slice {
		return errors.New("ScanAll: ts must be a pointer to a slice")
	}

	sliceElemType := sliceVal.Elem().Type().Elem()
	isPtrSlice := false
	if sliceElemType.Kind() == reflect.Ptr {
		isPtrSlice = true
		sliceElemType = sliceElemType.Elem()
	}
	if sliceElemType.Kind() != reflect.Struct {
		return errors.New("ScanAll: slice element must be struct or pointer to struct")
	}

	// 1. 获取列名
	cols := rows.FieldDescriptions()
	if len(cols) == 0 {
		return nil
	}
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = string(c.Name)
	}

	// 2. 【性能优化关键】在循环外预计算 "列索引 -> 字段索引" 的映射
	// fieldMap[i] = -1 表示该列不需要映射，否则表示对应 struct 的第几个字段
	fieldMap := getColToFieldMap(sliceElemType, colNames)

	resultSlice := reflect.MakeSlice(sliceVal.Elem().Type(), 0, 0)

	// 3. 循环读取
	for rows.Next() {
		newElemPtr := reflect.New(sliceElemType)
		newElem := newElemPtr.Elem()

		scanTargets := make([]interface{}, len(cols))

		for i, fieldIdx := range fieldMap {
			if fieldIdx >= 0 {
				// 直接通过索引获取字段，不再遍历
				// 注意：这里必须加 .Interface()
				scanTargets[i] = newElem.Field(fieldIdx).Addr().Interface()
			} else {
				var dummy interface{}
				scanTargets[i] = &dummy
			}
		}

		if err := rows.Scan(scanTargets...); err != nil {
			return err
		}

		if isPtrSlice {
			resultSlice = reflect.Append(resultSlice, newElemPtr)
		} else {
			resultSlice = reflect.Append(resultSlice, newElem)
		}
	}

	sliceVal.Elem().Set(resultSlice)
	return nil
}

// ScanOne 扫描单行
func (p *ScanAPI) ScanOne(t interface{}, rows pgx.Rows) error {
	if t == nil || rows == nil {
		return errors.New("ScanOne: t or rows is nil")
	}

	tVal := reflect.ValueOf(t)
	if tVal.Kind() != reflect.Ptr || tVal.Elem().Kind() != reflect.Struct {
		return errors.New("ScanOne: t must be a pointer to struct")
	}

	elem := tVal.Elem()
	elemType := elem.Type()

	cols := rows.FieldDescriptions()
	if len(cols) == 0 {
		return errors.New("ScanOne: no columns in rows")
	}

	if !rows.Next() {
		return errors.New("ScanOne: no rows to scan")
	}

	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = string(c.Name)
	}

	// 复用映射逻辑
	fieldMap := getColToFieldMap(elemType, colNames)
	scanTargets := make([]interface{}, len(cols))

	for i, fieldIdx := range fieldMap {
		if fieldIdx >= 0 {
			scanTargets[i] = elem.Field(fieldIdx).Addr().Interface()
		} else {
			var dummy interface{}
			scanTargets[i] = &dummy
		}
	}

	return rows.Scan(scanTargets...)
}

// getColToFieldMap 预计算列名到字段索引的映射
// 返回的 slice 长度等于 cols 长度，值为字段在 struct 中的 index，-1 表示未匹配
func getColToFieldMap(t reflect.Type, cols []string) []int {
	mapping := make([]int, len(cols))

	// 预先缓存 Struct 的所有有效字段信息，避免双重循环
	// key: tag或name, value: fieldIndex
	structFields := make(map[string]int)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// 【坑点规避】跳过未导出（私有）字段，否则 Interface() 会 panic
		if f.PkgPath != "" {
			continue
		}

		// 解析 Tag
		dbTag := strings.Split(f.Tag.Get("db"), ",")[0]
		jsonTag := strings.Split(f.Tag.Get("json"), ",")[0]

		if dbTag == "-" || jsonTag == "-" {
			continue
		}

		// 优先级 1: db tag
		if dbTag != "" {
			structFields[dbTag] = i
		}
		// 优先级 2: json tag (如果没有 db tag)
		if jsonTag != "" && dbTag == "" {
			// 只有在没有 db tag 冲突的情况下才记录 json tag
			if _, exists := structFields[jsonTag]; !exists {
				structFields[jsonTag] = i
			}
		}

		// 记录原始字段名用于兜底匹配
		if _, exists := structFields[f.Name]; !exists {
			structFields[f.Name] = i
		}
	}

	// 建立映射
	for i, colName := range cols {
		mapping[i] = -1 // 默认未匹配

		// 1. 精确匹配 (Tag 或 FieldName)
		if idx, ok := structFields[colName]; ok {
			mapping[i] = idx
			continue
		}

		// 2. 模糊匹配 (忽略大小写，解决 created_at vs CreatedAt 问题)
		// 这一步虽然是 O(N)，但只在 Setup 阶段做一次，可以接受
		for name, idx := range structFields {
			// 简单的规则：数据库列名移除下划线后，不区分大小写匹配
			// 例: user_id -> userid == UserID (Fold match)
			cleanCol := strings.ReplaceAll(colName, "_", "")
			if strings.EqualFold(cleanCol, name) {
				mapping[i] = idx
				break
			}
		}
	}

	return mapping
}
