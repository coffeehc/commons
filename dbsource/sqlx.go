package dbsource

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/coffeehc/base/errors"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// Although the NameMapper is convenient, in practice it should not
// be relied on except for application code.  If you are writing a library
// that uses sqlx, you should be aware that the name mappings you expect
// can be overridden by your user's application.

// NameMapper is used to map column names to struct field names.  By default,
// it uses strings.ToLower to lowercase struct field names.  It can be set
// to whatever you want, but it is encouraged to be set before sqlx is used
// as name-to-field mappings are cached after first use on a type.
var NameMapper = strings.ToLower
var origMapper = reflect.ValueOf(NameMapper)

// Rather than creating on init, this is created when necessary so that
// importers have time to customize the NameMapper.
var mpr *reflectx.Mapper

// mprMu protects mpr.
var mprMu sync.Mutex

// mapper returns a valid mapper using the configured NameMapper func.
func mapper() *reflectx.Mapper {
	mprMu.Lock()
	defer mprMu.Unlock()

	if mpr == nil {
		mpr = reflectx.NewMapperFunc("db", NameMapper)
	} else if origMapper != reflect.ValueOf(NameMapper) {
		// if NameMapper has changed, create a new mapper
		mpr = reflectx.NewMapperFunc("db", NameMapper)
		origMapper = reflect.ValueOf(NameMapper)
	}
	return mpr
}

// isScannable takes the reflect.Type and the actual dest value and returns
// whether or not it's Scannable.  Something is scannable if:
//   - it is not a struct
//   - it implements sql.Scanner
//   - it has no exported fields
func isScannable(t reflect.Type) bool {
	if reflect.PtrTo(t).Implements(_scannerInterface) {
		return true
	}
	if t.Kind() != reflect.Struct {
		return true
	}

	// it's not important that we use the right mapper for this particular object,
	// we're only concerned on how many exported fields this struct has
	m := mapper()
	if len(m.TypeMap(t).Index) == 0 {
		return true
	}
	return false
}

// determine if any of our extensions are unsafe
func isUnsafe(i interface{}) bool {
	// switch v := i.(type) {
	// case sqlx.Row:
	// 	return v.unsafe
	// case *sqlx.Row:
	// 	return v.unsafe
	// case sqlx.Rows:
	// 	return v.unsafe
	// case *sqlx.Rows:
	// 	return v.unsafe
	// case sqlx.NamedStmt:
	// 	return v.Stmt.unsafe
	// case *sqlx.NamedStmt:
	// 	return v.Stmt.unsafe
	// case sqlx.Stmt:
	// 	return v.unsafe
	// case *sqlx.Stmt:
	// 	return v.unsafe
	// case qStmt:
	// 	return v.unsafe
	// case *qStmt:
	// 	return v.unsafe
	// case sqlx.DB:
	// 	return v.unsafe
	// case *sqlx.DB:
	// 	return v.unsafe
	// case sqlx.Tx:
	// 	return v.unsafe
	// case *sqlx.Tx:
	// 	return v.unsafe
	// case sql.Rows, *sql.Rows:
	// 	return false
	// default:
	// 	return false
	// }
	return false
}

func mapperFor(i interface{}) *reflectx.Mapper {
	switch i.(type) {
	case sqlx.DB:
		return i.(sqlx.DB).Mapper
	case *sqlx.DB:
		return i.(*sqlx.DB).Mapper
	case sqlx.Tx:
		return i.(sqlx.Tx).Mapper
	case *sqlx.Tx:
		return i.(*sqlx.Tx).Mapper
	default:
		return mapper()
	}
}

var _scannerInterface = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
var _valuerInterface = reflect.TypeOf((*driver.Valuer)(nil)).Elem()

type rowsi interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(...interface{}) error
}

// structOnlyError returns an error appropriate for type when a non-scannable
// struct is expected but something else is given
func structOnlyError(t reflect.Type) error {
	isStruct := t.Kind() == reflect.Struct
	isScanner := reflect.PtrTo(t).Implements(_scannerInterface)
	if !isStruct {
		return errors.SystemError(fmt.Sprintf("expected %s but got %s", reflect.Struct, t.Kind()))
	}
	if isScanner {
		return errors.SystemError(fmt.Sprintf("structscan expects a struct dest but the provided struct type %s implements scanner", t.Name()))
	}
	return errors.SystemError(fmt.Sprintf("expected a struct, but struct %s has no exported fields", t.Name()))
}

// scanAll scans all rows into a destination, which must be a slice of any
// type.  If the destination slice type is a Struct, then StructScan will be
// used on each row.  If the destination is some other kind of base type, then
// each row must only have one column which can scan into that type.  This
// allows you to do something like:
//
//	rows, _ := db.Query("select id from people;")
//	var ids []int
//	scanAll(rows, &ids, false)
//
// and ids will be a list of the id results.  I realize that this is a desirable
// interface to expose to users, but for now it will only be exposed via changes
// to `Get` and `Select`.  The reason that this has been implemented like this is
// this is the only way to not duplicate reflect work in the new API while
// maintaining backwards compatibility.
func scanAll(rows rowsi, dest interface{}, structOnly bool) error {
	defer rows.Close()
	var v, vp reflect.Value

	value := reflect.ValueOf(dest)

	// json.Unmarshal returns errors for these
	if value.Kind() != reflect.Ptr {
		return errors.SystemError("must pass a pointer, not a value, to StructScan destination")
	}
	if value.IsNil() {
		return errors.SystemError("nil pointer passed to StructScan destination")
	}
	direct := reflect.Indirect(value)
	var err error
	slice, err := baseType(value.Type(), reflect.Slice)
	if err != nil {
		return errors.ConverError(err)
	}

	isPtr := slice.Elem().Kind() == reflect.Ptr
	base := reflectx.Deref(slice.Elem())
	scannable := isScannable(base)

	if structOnly && scannable {
		return structOnlyError(base)
	}

	columns, err := rows.Columns()
	if err != nil {
		return errors.ConverError(err)
	}

	// if it's a base type make sure it only has 1 column;  if not return an error
	if scannable && len(columns) > 1 {
		return errors.SystemError(fmt.Sprintf("non-struct dest type %s with >1 columns (%d)", base.Kind(), len(columns)))
	}

	if !scannable {
		var values []interface{}
		var m *reflectx.Mapper

		switch rows.(type) {
		case *sqlx.Rows:
			m = rows.(*sqlx.Rows).Mapper
		default:
			m = mapper()
		}

		fields := m.TraversalsByName(base, columns)
		// if we are not unsafe and are missing fields, return an error
		if f, err := missingFields(fields); err != nil && !isUnsafe(rows) {
			return errors.SystemError(fmt.Sprintf("missing destination name %s in %T", columns[f], dest))
		}
		values = make([]interface{}, len(columns))
		for rows.Next() {
			// create a new struct type (which returns PtrTo) and indirect it
			vp = reflect.New(base)
			v = reflect.Indirect(vp)

			err = fieldsByTraversal(v, fields, values, true)
			if err != nil {
				return errors.ConverError(err)
			}
			// scan into the struct field pointers and append to our results
			err = rows.Scan(values...)
			if err != nil {
				return errors.ConverError(err)
			}
			if isPtr {
				direct.Set(reflect.Append(direct, vp))
			} else {
				direct.Set(reflect.Append(direct, v))
			}
		}
	} else {
		for rows.Next() {
			vp = reflect.New(base)
			err = rows.Scan(vp.Interface())
			if err != nil {
				return errors.ConverError(err)
			}
			// append
			if isPtr {
				direct.Set(reflect.Append(direct, vp))
			} else {
				direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
			}
		}
	}
	if rows.Err() != nil {
		return errors.ConverError(rows.Err())
	}
	return nil
}

// reflect helpers

func baseType(t reflect.Type, expected reflect.Kind) (reflect.Type, error) {
	t = reflectx.Deref(t)
	if t.Kind() != expected {
		return nil, errors.SystemError(fmt.Sprintf("expected %s but got %s", expected, t.Kind()))
	}
	return t, nil
}

// fieldsByName fills a values interface with fields from the passed value based
// on the traversals in int.  If ptrs is true, return addresses instead of values.
// We write this instead of using FieldsByName to save allocations and map lookups
// when iterating over many rows.  Empty traversals will get an interface pointer.
// Because of the necessity of requesting ptrs or values, it's considered a bit too
// specialized for inclusion in reflectx itself.
func fieldsByTraversal(v reflect.Value, traversals [][]int, values []interface{}, ptrs bool) error {
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return errors.SystemError("argument not a struct")
	}

	for i, traversal := range traversals {
		if len(traversal) == 0 {
			values[i] = new(interface{})
			continue
		}
		f := reflectx.FieldByIndexes(v, traversal)
		if ptrs {
			values[i] = f.Addr().Interface()
		} else {
			values[i] = f.Interface()
		}
	}
	return nil
}

func missingFields(transversals [][]int) (field int, err error) {
	for i, t := range transversals {
		if len(t) == 0 {
			return i, errors.SystemError("missing field")
		}
	}
	return 0, nil
}
