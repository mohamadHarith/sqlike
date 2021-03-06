package sqlike

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/si3nloong/sqlike/reflext"
	sqldialect "github.com/si3nloong/sqlike/sql/dialect"
	sqldriver "github.com/si3nloong/sqlike/sql/driver"
	"github.com/si3nloong/sqlike/sql/expr"
	sqlstmt "github.com/si3nloong/sqlike/sql/stmt"
	"github.com/si3nloong/sqlike/sqlike/actions"
	"github.com/si3nloong/sqlike/sqlike/logs"
	"github.com/si3nloong/sqlike/sqlike/options"
)

// DestroyOne : hard delete a record on the table using primary key. You should alway have primary key defined in your struct in order to use this api.
func (tb *Table) DestroyOne(ctx context.Context, delete interface{}) error {
	return destroyOne(
		ctx,
		tb.dbName,
		tb.name,
		tb.pk,
		tb.client.cache,
		tb.driver,
		tb.dialect,
		tb.logger,
		delete,
	)
}

// DeleteOne : delete single record on the table using where clause.
func (tb *Table) DeleteOne(ctx context.Context, act actions.DeleteOneStatement, opts ...*options.DeleteOneOptions) (int64, error) {
	x := new(actions.DeleteOneActions)
	if act != nil {
		*x = *(act.(*actions.DeleteOneActions))
	}
	opt := new(options.DeleteOneOptions)
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}
	x.Limit(1)
	return deleteMany(
		ctx,
		tb.dbName,
		tb.name,
		tb.driver,
		tb.dialect,
		tb.logger,
		&x.DeleteActions,
		&opt.DeleteOptions,
	)
}

// Delete : delete multiple record on the table using where clause. If you didn't provided any where clause, it will throw error. For multiple record deletion without where clause, you should use `Truncate` instead.
func (tb *Table) Delete(ctx context.Context, act actions.DeleteStatement, opts ...*options.DeleteOptions) (int64, error) {
	x := new(actions.DeleteActions)
	if act != nil {
		*x = *(act.(*actions.DeleteActions))
	}
	opt := new(options.DeleteOptions)
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}
	return deleteMany(
		ctx,
		tb.dbName,
		tb.name,
		tb.driver,
		tb.dialect,
		tb.logger,
		x,
		opt,
	)
}

func deleteMany(ctx context.Context, dbName, tbName string, driver sqldriver.Driver, dialect sqldialect.Dialect, logger logs.Logger, act *actions.DeleteActions, opt *options.DeleteOptions) (int64, error) {
	if act.Database == "" {
		act.Database = dbName
	}
	if act.Table == "" {
		act.Table = tbName
	}
	if len(act.Conditions) < 1 {
		return 0, errors.New("sqlike: empty condition is not allow for delete, please use truncate instead")
	}

	stmt := sqlstmt.AcquireStmt(dialect)
	defer sqlstmt.ReleaseStmt(stmt)
	if err := dialect.Delete(stmt, act); err != nil {
		return 0, err
	}
	result, err := sqldriver.Execute(
		ctx,
		driver,
		stmt,
		logger,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func destroyOne(ctx context.Context, dbName, tbName, pk string, cache reflext.StructMapper, driver sqldriver.Driver, dialect sqldialect.Dialect, logger logs.Logger, delete interface{}) error {
	v := reflext.ValueOf(delete)
	if !v.IsValid() {
		return ErrInvalidInput
	}
	t := v.Type()
	if !reflext.IsKind(t, reflect.Ptr) {
		return ErrUnaddressableEntity
	}
	if v.IsNil() {
		return ErrNilEntity
	}

	cdc := cache.CodecByType(t)
	f, exists := cdc.LookUpFieldByName(pk)
	if !exists {
		return fmt.Errorf("sqlike: missing primary key field %q", pk)
	}

	x := new(actions.DeleteActions)
	x.Database = dbName
	x.Table = tbName
	fv := cache.FieldByIndexesReadOnly(v, f.Index())
	x.Where(expr.Equal(f.Name(), fv.Interface()))
	x.Limit(1)

	stmt := sqlstmt.AcquireStmt(dialect)
	defer sqlstmt.ReleaseStmt(stmt)
	if err := dialect.Delete(stmt, x); err != nil {
		return err
	}
	result, err := sqldriver.Execute(
		ctx,
		driver,
		stmt,
		logger,
	)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected <= 0 {
		return errors.New("sqlike: unable to delete entity")
	}
	return err
}
