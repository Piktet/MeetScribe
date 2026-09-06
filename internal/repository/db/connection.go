// Package db — взаимодействие с базой данных PostgreSQL.
package db

import (
	"context"
	"database/sql"

	"github.com/Piktet/MeetScribe/internal/logger"
	"github.com/Piktet/MeetScribe/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DBConnection — подключение к PostgreSQL.
// Реализует интерфейс model.Connection.
type DBConnection struct {
	conn *sql.DB
}

// rowsAdapter — адаптер sql.Rows к model.Rows.
type rowsAdapter struct {
	rows *sql.Rows
}

func (r *rowsAdapter) Close() error               { return r.rows.Close() }
func (r *rowsAdapter) Columns() ([]string, error) { return r.rows.Columns() }
func (r *rowsAdapter) Next() bool                 { return r.rows.Next() }
func (r *rowsAdapter) Scan(dest ...any) error     { return r.rows.Scan(dest...) }
func (r *rowsAdapter) Err() error                 { return r.rows.Err() }

// txAdapter — адаптер sql.Tx к model.Tx.
type txAdapter struct {
	tx *sql.Tx
}

func (t *txAdapter) Commit() error   { return t.tx.Commit() }
func (t *txAdapter) Rollback() error { return t.tx.Rollback() }

// New создает новое подключение к PostgreSQL.
// s — строка подключения (DSN). Возвращает ошибку при неудачном подключении.
func New(s string) (*DBConnection, error) {
	conn, err := sql.Open("pgx", s)
	if err != nil {
		logger.Error(err, "error create connection")
		return nil, err
	}
	return &DBConnection{conn: conn}, nil
}

// Query выполняет SELECT-запрос к базе данных.
func (p *DBConnection) Query(ctx context.Context, q string, args ...any) (model.Rows, error) {
	rows, err := p.conn.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return &rowsAdapter{rows: rows}, nil
}

// Execute выполняет команду (INSERT, UPDATE, DELETE, CREATE и т.д.).
func (p *DBConnection) Execute(ctx context.Context, q string, args ...any) error {
	_, err := p.conn.ExecContext(ctx, q, args...)
	return err
}

// BeginTx начинает новую транзакцию.
func (p *DBConnection) BeginTx(ctx context.Context) (model.Tx, error) {
	tx, err := p.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &txAdapter{tx: tx}, nil
}
