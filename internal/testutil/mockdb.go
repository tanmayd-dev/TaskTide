package testutil

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MockDB struct {
	QueryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	ExecFunc     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return RowFromError(pgx.ErrNoRows)
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, args...)
	}
	return nil, nil
}

type MockRow struct {
	scanFunc func(dest ...any) error
}

func (r *MockRow) Scan(dest ...any) error {
	return r.scanFunc(dest...)
}

func RowFromValues(values ...any) *MockRow {
	return &MockRow{
		scanFunc: func(dest ...any) error {
			if len(dest) != len(values) {
				return fmt.Errorf("expected %d scan destinations, got %d", len(values), len(dest))
			}
			for i := range dest {
				if dest[i] == nil {
					continue
				}
				rv := reflect.ValueOf(dest[i])
				if rv.Kind() != reflect.Ptr || rv.IsNil() {
					return fmt.Errorf("scan destination %d is not a non-nil pointer", i)
				}
				value := reflect.ValueOf(values[i])
				if !value.IsValid() {
					rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
					continue
				}
				if !value.Type().AssignableTo(rv.Elem().Type()) {
					if value.Type().ConvertibleTo(rv.Elem().Type()) {
						value = value.Convert(rv.Elem().Type())
					} else {
						return fmt.Errorf("cannot assign value of type %T to destination %T", values[i], dest[i])
					}
				}
				rv.Elem().Set(value)
			}
			return nil
		},
	}
}

func RowFromError(err error) *MockRow {
	return &MockRow{
		scanFunc: func(dest ...any) error {
			return err
		},
	}
}
