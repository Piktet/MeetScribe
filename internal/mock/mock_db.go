package mock

import (
	"context"
	"sync"

	"github.com/Piktet/MeetScribe/internal/model"
)

// MockDB — заглушка для model.Connection (база данных).
// Позволяет настраивать поведение методов и проверять вызовы.
type MockDB struct {
	mx sync.Mutex

	// Ожидаемые значения для Query
	QueryRows  model.Rows
	QueryError error

	// Ожидаемые значения для Execute
	ExecuteError error

	// Ожидаемые значения для BeginTx
	Tx           model.Tx
	BeginTxError error

	// Счётчики вызовов
	QueryCalls   int
	ExecuteCalls int
	BeginTxCalls int

	// Записанные аргументы
	QuerySQL    string
	QueryArgs   []any
	ExecuteSQL  string
	ExecuteArgs []any
}

// mockRows — заглушка для model.Rows.
type mockRows struct {
	closed   bool
	rowsData [][]any
	current  int
}

var _ model.Rows = (*mockRows)(nil)

func (r *mockRows) Close() error {
	r.closed = true
	return nil
}

func (r *mockRows) Columns() ([]string, error) {
	return []string{}, nil
}

func (r *mockRows) Next() bool {
	if r.closed {
		return false
	}
	if r.current < len(r.rowsData) {
		r.current++
		return true
	}
	return false
}

func (r *mockRows) Scan(dest ...any) error {
	if r.current == 0 || r.current > len(r.rowsData) {
		return nil
	}
	row := r.rowsData[r.current-1]
	for i := range dest {
		if i < len(row) {
			switch d := dest[i].(type) {
			case *string:
				if v, ok := row[i].(string); ok {
					*d = v
				}
			case *int64:
				if v, ok := row[i].(int64); ok {
					*d = v
				}
			default:
				// Ignoring type mismatch for simplicity
			}
		}
	}
	return nil
}

func (r *mockRows) Err() error {
	return nil
}

// mockTx — заглушка для model.Tx.
type mockTx struct {
	closed bool
}

var _ model.Tx = (*mockTx)(nil)

func (t *mockTx) Commit() error {
	t.closed = true
	return nil
}

func (t *mockTx) Rollback() error {
	t.closed = true
	return nil
}

// Compile-time check: проверяем, что MockDB реализует model.Connection.
var _ model.Connection = (*MockDB)(nil)

// Query — заглушка метода Query.
// Возвращает заданные строки и ошибку. Записывает SQL-запрос и аргументы.
func (m *MockDB) Query(ctx context.Context, query string, args ...any) (model.Rows, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.QueryCalls++
	m.QuerySQL = query
	m.QueryArgs = args

	if m.QueryError != nil {
		return nil, m.QueryError
	}
	if m.QueryRows == nil {
		return nil, nil
	}
	return m.QueryRows, nil
}

// Execute — заглушка метода Execute.
// Возвращает заданную ошибку. Записывает SQL-команду и аргументы.
func (m *MockDB) Execute(ctx context.Context, query string, args ...any) error {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.ExecuteCalls++
	m.ExecuteSQL = query
	m.ExecuteArgs = args

	return m.ExecuteError
}

// BeginTx — заглушка метода BeginTx.
// Возвращает заданный tx и ошибку.
func (m *MockDB) BeginTx(ctx context.Context) (model.Tx, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.BeginTxCalls++

	if m.BeginTxError != nil {
		return nil, m.BeginTxError
	}
	if m.Tx == nil {
		m.Tx = &mockTx{}
	}
	return m.Tx, nil
}

// Reset сбрасывает все счётчики и записанные данные.
func (m *MockDB) Reset() {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.QueryCalls = 0
	m.ExecuteCalls = 0
	m.BeginTxCalls = 0
	m.QuerySQL = ""
	m.QueryArgs = nil
	m.ExecuteSQL = ""
	m.ExecuteArgs = nil
}
