package mock

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Piktet/MeetScribe/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockDB_Query(t *testing.T) {
	ctx := context.Background()

	t.Run("with rows", func(t *testing.T) {
		m := &MockDB{QueryRows: &mockRows{rowsData: [][]any{{"1", "name"}}}}
		rows, err := m.Query(ctx, "select 1", 42)
		require.NoError(t, err)
		require.NotNil(t, rows)
		assert.Equal(t, 1, m.QueryCalls)
		assert.Equal(t, "select 1", m.QuerySQL)
		assert.Equal(t, []any{42}, m.QueryArgs)

		// Проверяем поведение mockRows.
		assert.True(t, rows.Next())
		var id string
		var name string
		require.NoError(t, rows.Scan(&id, &name))
		assert.Equal(t, "1", id)
		assert.Equal(t, "name", name)
		assert.False(t, rows.Next())
		assert.NoError(t, rows.Err())
		cols, err := rows.Columns()
		require.NoError(t, err)
		assert.Empty(t, cols)
		require.NoError(t, rows.Close())
	})

	t.Run("query error", func(t *testing.T) {
		m := &MockDB{QueryError: errors.New("db error")}
		rows, err := m.Query(ctx, "select 1")
		assert.Error(t, err)
		assert.Nil(t, rows)
	})

	t.Run("no rows configured", func(t *testing.T) {
		m := &MockDB{}
		rows, err := m.Query(ctx, "select 1")
		require.NoError(t, err)
		assert.Nil(t, rows)
	})
}

func TestMockDB_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &MockDB{}
		require.NoError(t, m.Execute(ctx, "insert into t values($1)", "a"))
		assert.Equal(t, 1, m.ExecuteCalls)
		assert.Equal(t, "insert into t values($1)", m.ExecuteSQL)
		assert.Equal(t, []any{"a"}, m.ExecuteArgs)
	})

	t.Run("error", func(t *testing.T) {
		m := &MockDB{ExecuteError: errors.New("db error")}
		assert.Error(t, m.Execute(ctx, "insert into t"))
	})
}

func TestMockDB_BeginTx(t *testing.T) {
	ctx := context.Background()

	t.Run("default tx", func(t *testing.T) {
		m := &MockDB{}
		tx, err := m.BeginTx(ctx)
		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.Equal(t, 1, m.BeginTxCalls)
		require.NoError(t, tx.Commit())
	})

	t.Run("custom tx rollback", func(t *testing.T) {
		tx := &mockTx{}
		m := &MockDB{Tx: tx}
		got, err := m.BeginTx(ctx)
		require.NoError(t, err)
		require.NoError(t, got.Rollback())
		assert.True(t, tx.closed)
	})

	t.Run("error", func(t *testing.T) {
		m := &MockDB{BeginTxError: errors.New("tx error")}
		tx, err := m.BeginTx(ctx)
		assert.Error(t, err)
		assert.Nil(t, tx)
	})
}

func TestMockDB_Reset(t *testing.T) {
	ctx := context.Background()
	m := &MockDB{}
	_, _ = m.Query(ctx, "select 1", 1)
	_ = m.Execute(ctx, "insert into t")
	_, _ = m.BeginTx(ctx)

	m.Reset()

	assert.Zero(t, m.QueryCalls)
	assert.Zero(t, m.ExecuteCalls)
	assert.Zero(t, m.BeginTxCalls)
	assert.Empty(t, m.QuerySQL)
	assert.Nil(t, m.QueryArgs)
	assert.Empty(t, m.ExecuteSQL)
	assert.Nil(t, m.ExecuteArgs)
}

func TestMockRows_ScanUnsupportedType(t *testing.T) {
	r := &mockRows{rowsData: [][]any{{"1"}}}
	require.True(t, r.Next())
	// Неподдерживаемый тип назначения игнорируется.
	var b bool
	require.NoError(t, r.Scan(&b))
	assert.False(t, b)
}

func TestMockRows_ScanOutOfRange(t *testing.T) {
	r := &mockRows{rowsData: [][]any{{"1"}}}
	require.True(t, r.Next())
	// Сканирование до вызова Next не должно паниковать.
	var s string
	r2 := &mockRows{rowsData: [][]any{{"1"}}}
	require.NoError(t, r2.Scan(&s))
	assert.Empty(t, s)
	require.NoError(t, r.Scan(&s))
	assert.Equal(t, "1", s)
}

func TestMockRows_ScanAfterClose(t *testing.T) {
	r := &mockRows{rowsData: [][]any{{"1"}}}
	require.NoError(t, r.Close())
	assert.True(t, r.closed)
	assert.False(t, r.Next())
}

func TestMockRows_ScanTypeMismatch(t *testing.T) {
	r := &mockRows{rowsData: [][]any{{"1"}}}
	require.True(t, r.Next())
	var n int64
	// Значение строка, назначение int64 — несовпадение типов игнорируется.
	require.NoError(t, r.Scan(&n))
	assert.Zero(t, n)
}

func TestMockTx(t *testing.T) {
	tx := &mockTx{}
	require.NoError(t, tx.Commit())
	assert.True(t, tx.closed)
	tx2 := &mockTx{}
	require.NoError(t, tx2.Rollback())
	assert.True(t, tx2.closed)
}

func TestMockLLMClient_GetShort(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &MockLLMClient{GetShortResult: "выжимка"}
		result, err := m.GetShort(ctx, []byte("текст"))
		require.NoError(t, err)
		assert.Equal(t, "выжимка", result)
		assert.Equal(t, 1, m.GetShortCalls)
		assert.Equal(t, []byte("текст"), m.GetShortInput)
	})

	t.Run("error", func(t *testing.T) {
		m := &MockLLMClient{GetShortError: errors.New("llm error")}
		_, err := m.GetShort(ctx, []byte("текст"))
		assert.Error(t, err)
	})
}

func TestMockLLMClient_GetAnswer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &MockLLMClient{GetAnswerResult: "ответ"}
		result, err := m.GetAnswer(ctx, "вопрос", "контекст")
		require.NoError(t, err)
		assert.Equal(t, "ответ", result)
		assert.Equal(t, 1, m.GetAnswerCalls)
		assert.Equal(t, "вопрос", m.GetAnswerPrompt)
		assert.Equal(t, "контекст", m.GetAnswerCtx)
	})

	t.Run("error", func(t *testing.T) {
		m := &MockLLMClient{GetAnswerError: errors.New("llm error")}
		_, err := m.GetAnswer(ctx, "вопрос", "контекст")
		assert.Error(t, err)
	})
}

func TestMockLLMClient_Reset(t *testing.T) {
	ctx := context.Background()
	m := &MockLLMClient{GetShortResult: "a", GetAnswerResult: "b"}
	_, _ = m.GetShort(ctx, []byte("text"))
	_, _ = m.GetAnswer(ctx, "p", "c")

	m.Reset()

	assert.Zero(t, m.GetShortCalls)
	assert.Zero(t, m.GetAnswerCalls)
	assert.Nil(t, m.GetShortInput)
	assert.Empty(t, m.GetAnswerPrompt)
	assert.Empty(t, m.GetAnswerCtx)
}

// TestMockLLMClient_ImplementsModelInterface проверяет совместимость с model.LLMClient.
func TestMockLLMClient_ImplementsModelInterface(t *testing.T) {
	var _ model.LLMClient = (*MockLLMClient)(nil)
	var _ model.Connection = (*MockDB)(nil)
	var _ model.Rows = (*mockRows)(nil)
	var _ model.Tx = (*mockTx)(nil)
	_ = strings.NewReader // держим импорт strings используемым
}
