package dialect

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx/reflectx"
)

var scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

// ScanAll maps every row into dest using mapper semantics compatible with sqlx Select.
func ScanAll(rows Rows, dest any, mapper *reflectx.Mapper) error {
	destination := reflect.ValueOf(dest)
	if destination.Kind() != reflect.Ptr || destination.IsNil() {
		return fmt.Errorf("查询目标必须是非空指针")
	}
	slice := destination.Elem()
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("多行查询目标必须是切片指针")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	slice.SetLen(0)
	elementType := slice.Type().Elem()
	pointerElement := elementType.Kind() == reflect.Ptr
	baseType := reflectx.Deref(elementType)
	scannable := isScannable(baseType, mapper)
	if scannable && len(columns) != 1 {
		// pgx may expose no columns until Next surfaces a canceled query's error.
		// This result cannot be scanned as a scalar anyway; prefer its actual
		// terminal error over a misleading column-count failure.
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return err
			}
		}
		return fmt.Errorf("标量查询目标要求一列，实际返回 %d 列", len(columns))
	}

	var traversals [][]int
	if !scannable {
		traversals = mapper.TraversalsByName(baseType, columns)
		if missing := missingTraversal(traversals); missing >= 0 {
			return fmt.Errorf("查询列 %s 在目标 %T 中没有对应字段", columns[missing], dest)
		}
	}

	for rows.Next() {
		elementPointer := reflect.New(baseType)
		if scannable {
			if err := rows.Scan(elementPointer.Interface()); err != nil {
				return err
			}
		} else {
			values := traversalTargets(elementPointer.Elem(), traversals)
			if err := rows.Scan(values...); err != nil {
				return err
			}
		}
		if pointerElement {
			slice.Set(reflect.Append(slice, elementPointer))
		} else {
			slice.Set(reflect.Append(slice, elementPointer.Elem()))
		}
	}
	return rows.Err()
}

// ScanOne maps the first row into dest using mapper semantics compatible with sqlx Get.
func ScanOne(rows Rows, dest any, mapper *reflectx.Mapper) error {
	destination := reflect.ValueOf(dest)
	if destination.Kind() != reflect.Ptr || destination.IsNil() {
		return fmt.Errorf("查询目标必须是非空指针")
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	baseType := reflectx.Deref(destination.Type())
	if isScannable(baseType, mapper) {
		if len(columns) != 1 {
			return fmt.Errorf("标量查询目标要求一列，实际返回 %d 列", len(columns))
		}
		return rows.Scan(dest)
	}
	traversals := mapper.TraversalsByName(baseType, columns)
	if missing := missingTraversal(traversals); missing >= 0 {
		return fmt.Errorf("查询列 %s 在目标 %T 中没有对应字段", columns[missing], dest)
	}
	return rows.Scan(traversalTargets(destination.Elem(), traversals)...)
}

func isScannable(valueType reflect.Type, mapper *reflectx.Mapper) bool {
	if reflect.PtrTo(valueType).Implements(scannerType) {
		return true
	}
	if valueType.Kind() != reflect.Struct {
		return true
	}
	return len(mapper.TypeMap(valueType).Index) == 0
}

func missingTraversal(traversals [][]int) int {
	for index, traversal := range traversals {
		if len(traversal) == 0 {
			return index
		}
	}
	return -1
}

func traversalTargets(value reflect.Value, traversals [][]int) []any {
	targets := make([]any, len(traversals))
	for index, traversal := range traversals {
		targets[index] = reflectx.FieldByIndexes(value, traversal).Addr().Interface()
	}
	return targets
}
