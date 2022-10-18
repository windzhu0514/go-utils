package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"entgo.io/ent/examples/o2mrecur/ent"
	"github.com/jmoiron/sqlx"
)

func DBFields(rv reflect.Value) []string {
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	rt := rv.Type()

	var fields []string
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			sf := rv.Field(i)
			if sf.Kind() == reflect.Struct {
				fields = append(fields, DBFields(sf)...)
				continue
			}

			tagName := rt.Field(i).Tag.Get("db")
			if tagName != "" {
				fields = append(fields, tagName)
			}
		}
		return fields
	}

	if rv.Kind() == reflect.Map {
		for _, key := range rv.MapKeys() {
			fields = append(fields, key.String())
		}
		return fields
	}

	panic(fmt.Errorf("dbFields requires a struct or a map, found: %s", rv.Kind().String()))
}

func WithTxEnt(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) (err error) {
	var tx *ent.Tx
	tx, err = client.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("%v", v)
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("%w:%v", err, rerr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("rolling back transaction: %w", rerr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func WithTxSqlx(db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("%v", v)
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("%w:%v", err, rerr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("rolling back transaction: %w", rerr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func WithTx(db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("%v", v)
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("%w:%v", err, rerr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("rolling back transaction: %w", rerr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
