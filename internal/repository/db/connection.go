// Package db — взаимодействие с базой данных PostgreSQL.
package db

import (
	"context"
	"database/sql"

	"github.com/Piktet/tg_bot/internal/logger"

	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

// DBConnection — подключение к PostgreSQL.
// Реализует интерфейс model.Connection.
type DBConnection struct {
	conn *sql.DB
}

// New создает новое подключение к PostgreSQL.
// s — строка подключения (DSN). Возвращает ошибку при неудачном подключении.
func New(s string) (*DBConnection, error) {
	conn, err := sql.Open("pgx", s)
	if err != nil {
		logger.Log().Error("error create connection", zap.Error(err))
		return nil, err
	}
	return &DBConnection{conn: conn}, nil
}

// Query выполняет SELECT-запрос к базе данных.
func (p *DBConnection) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return p.conn.QueryContext(ctx, q, args...)

}

// Execute выполняет команду (INSERT, UPDATE, DELETE, CREATE и т.д.).
func (p *DBConnection) Execute(ctx context.Context, q string, args ...any) error {
	_, err := p.conn.ExecContext(ctx, q, args...)
	return err

}

// BeginTx начинает новую транзакцию.
func (p *DBConnection) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return p.conn.BeginTx(ctx, nil)

}
