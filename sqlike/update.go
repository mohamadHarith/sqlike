package sqlike

import (
	"context"
	"errors"
	"reflect"

	"github.com/si3nloong/sqlike/reflext"
	sqldialect "github.com/si3nloong/sqlike/sql/dialect"
	sqldriver "github.com/si3nloong/sqlike/sql/driver"
	"github.com/si3nloong/sqlike/sql/expr"
	"github.com/si3nloong/sqlike/sqlike/actions"
	"github.com/si3nloong/sqlike/sqlike/logs"
	"github.com/si3nloong/sqlike/sqlike/options"
)

// ModifyOne :
func (tb *Table) ModifyOne(update interface{}, opts ...*options.ModifyOneOptions) error {
	return modifyOne(
		context.Background(),
		tb.dbName,
		tb.name,
		tb.pk,
		tb.dialect,
		tb.driver,
		tb.logger,
		update,
		opts,
	)
}

func modifyOne(ctx context.Context, dbName, tbName, pk string, dialect sqldialect.Dialect, driver sqldriver.Driver, logger logs.Logger, update interface{}, opts []*options.ModifyOneOptions) error {
	v := reflext.ValueOf(update)
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

	mapper := reflext.DefaultMapper
	cdc := mapper.CodecByType(t)
	// if _, exists := cdc.Names[pk]; !exists {
	// 	return fmt.Errorf("sqlike: missing primary key field %q", pk)
	// }

	opt := new(options.ModifyOneOptions)
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

	fields := skipColumns(cdc.Properties, opt.Omits)
	x := new(actions.UpdateActions)
	x.Table = tbName

	var pkv = [2]interface{}{}
	for _, sf := range fields {
		fv := mapper.FieldByIndexesReadOnly(v, sf.Index)
		if _, ok := sf.Tag.LookUp("primary_key"); ok {
			if pkv[0] != nil {
				x.Set(expr.ColumnValue(pkv[0].(string), pkv[1]))
			}
			pkv[0] = sf.Path
			pkv[1] = fv.Interface()
			continue
		}
		if sf.Path == pk && pkv[0] == nil {
			pkv[0] = sf.Path
			pkv[1] = fv.Interface()
			continue
		}
		x.Set(expr.ColumnValue(sf.Path, fv.Interface()))
	}

	if pkv[0] == nil {
		return errors.New("sqlike: missing primary key field")
	}

	x.Where(expr.Equal(pkv[0], pkv[1]))
	x.Limit(1)
	x.Table = tbName
	x.Database = dbName
	stmt, err := dialect.Update(x)
	if err != nil {
		return err
	}

	result, err := sqldriver.Execute(
		ctx,
		driver,
		stmt,
		getLogger(logger, opt.Debug),
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected < 1 {
		return ErrNoRecordAffected
	}
	return nil
}

// UpdateOne :
func (tb *Table) UpdateOne(act actions.UpdateOneStatement, opts ...*options.UpdateOneOptions) (int64, error) {
	x := new(actions.UpdateOneActions)
	if act != nil {
		*x = *(act.(*actions.UpdateOneActions))
	}
	opt := new(options.UpdateOneOptions)
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

	x.Limit(1)
	return update(
		context.Background(),
		tb.dbName,
		tb.name,
		tb.driver,
		tb.dialect,
		tb.logger,
		&x.UpdateActions,
		&opt.UpdateOptions,
	)
}

// Update :
func (tb *Table) Update(act actions.UpdateStatement, opts ...*options.UpdateOptions) (int64, error) {
	x := new(actions.UpdateActions)
	if act != nil {
		*x = *(act.(*actions.UpdateActions))
	}
	opt := new(options.UpdateOptions)
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}
	return update(
		context.Background(),
		tb.dbName,
		tb.name,
		tb.driver,
		tb.dialect,
		tb.logger,
		x,
		opt,
	)
}

func update(ctx context.Context, dbName, tbName string, driver sqldriver.Driver, dialect sqldialect.Dialect, logger logs.Logger, act *actions.UpdateActions, opt *options.UpdateOptions) (int64, error) {
	if act.Database == "" {
		act.Database = dbName
	}
	if act.Table == "" {
		act.Table = tbName
	}
	if len(act.Values) < 1 {
		return 0, ErrNoValueUpdate
	}
	stmt, err := dialect.Update(act)
	if err != nil {
		return 0, err
	}
	result, err := sqldriver.Execute(
		ctx,
		driver,
		stmt,
		getLogger(logger, opt.Debug),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
