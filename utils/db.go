package utils

import (
	"context"
	"fmt"
	"reflect"

	"entgo.io/ent/examples/fs/ent"
)

// SQLXDBFields 获取结构体（tag:db）或map的字段名
func SQLXDBFields(values interface{}) []string {
	v := reflect.ValueOf(values)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var fields []string
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i).Tag.Get("db")
			if field != "" {
				fields = append(fields, field)
			}
		}
		return fields
	}

	if v.Kind() == reflect.Map {
		for _, keyv := range v.MapKeys() {
			fields = append(fields, keyv.String())
		}
		return fields
	}

	return nil
}

func WithEntTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) (err error) {
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
			err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
