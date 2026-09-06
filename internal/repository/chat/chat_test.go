package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newChatServer создает сервер, эмулирующий /api/v1/chat/completions.
// checkRequest вызывается с распарсенным телом запроса.
func newChatServer(t *testing.T, status int, body string, checkRequest func(t *testing.T, req ChatRequest)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/chat/completions", r.URL.Path)
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req ChatRequest
		require.NoError(t, json.Unmarshal(data, &req))
		if checkRequest != nil {
			checkRequest(t, req)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGetShort(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv := newChatServer(t, http.StatusOK,
			`{"choices":[{"message":{"content":"краткая выжимка"}}]}`,
			func(t *testing.T, req ChatRequest) {
				assert.Equal(t, "GigaChat", req.Model)
				require.Len(t, req.Messages, 1)
				assert.Equal(t, "user", req.Messages[0].Role)
				assert.Contains(t, req.Messages[0].Content, "исходный текст")
			})
		result, err := GetShort(ctx, srv.URL, "tok", []byte("исходный текст"))
		require.NoError(t, err)
		assert.Equal(t, "краткая выжимка", result)
	})

	t.Run("http error", func(t *testing.T) {
		srv := newChatServer(t, http.StatusInternalServerError, ``, nil)
		_, err := GetShort(ctx, srv.URL, "tok", []byte("text"))
		assert.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := newChatServer(t, http.StatusOK, `not-json`, nil)
		_, err := GetShort(ctx, srv.URL, "tok", []byte("text"))
		assert.Error(t, err)
	})

	t.Run("empty choices", func(t *testing.T) {
		srv := newChatServer(t, http.StatusOK, `{"choices":[]}`, nil)
		result, err := GetShort(ctx, srv.URL, "tok", []byte("text"))
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("unreachable host", func(t *testing.T) {
		_, err := GetShort(ctx, "http://127.0.0.1:1", "tok", []byte("text"))
		assert.Error(t, err)
	})
}

func TestGetAnswer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv := newChatServer(t, http.StatusOK,
			`{"choices":[{"message":{"content":"ответ на вопрос"}}]}`,
			func(t *testing.T, req ChatRequest) {
				require.Len(t, req.Messages, 1)
				assert.Contains(t, req.Messages[0].Content, "контекст встречи")
				assert.Contains(t, req.Messages[0].Content, "о чём встреча?")
			})
		result, err := GetAnswer(ctx, srv.URL, "tok", "о чём встреча?", "контекст встречи")
		require.NoError(t, err)
		assert.Equal(t, "ответ на вопрос", result)
	})

	t.Run("http error", func(t *testing.T) {
		srv := newChatServer(t, http.StatusForbidden, ``, nil)
		_, err := GetAnswer(ctx, srv.URL, "tok", "вопрос", "контекст")
		assert.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := newChatServer(t, http.StatusOK, `{`, nil)
		_, err := GetAnswer(ctx, srv.URL, "tok", "вопрос", "контекст")
		assert.Error(t, err)
	})
}
