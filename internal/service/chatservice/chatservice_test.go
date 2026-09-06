package chatservice_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Piktet/MeetScribe/internal/repository/chat"
	"github.com/Piktet/MeetScribe/internal/service/chatservice"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEnv создает тестовый сервер GigaChat и подключённый к нему ChatConnection.
func newEnv(t *testing.T, status int, body string) (*httptest.Server, *chat.ChatConnection) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/oauth":
			w.Write([]byte(`{"access_token":"tok-1","expires_at":4102444800}`))
		case "/api/v1/chat/completions":
			w.WriteHeader(status)
			w.Write([]byte(body))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	conn := chat.New(srv.URL, "uid", "key")
	require.NoError(t, conn.Connect(context.Background()))
	return srv, conn
}

func TestChatService_GetShort(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, conn := newEnv(t, http.StatusOK, `{"choices":[{"message":{"content":"выжимка"}}]}`)
		svc := chatservice.New(srv.URL, conn)
		result, err := svc.GetShort(ctx, []byte("текст встречи"))
		require.NoError(t, err)
		assert.Equal(t, "выжимка", result)
	})

	t.Run("api error", func(t *testing.T) {
		srv, conn := newEnv(t, http.StatusInternalServerError, ``)
		svc := chatservice.New(srv.URL, conn)
		_, err := svc.GetShort(ctx, []byte("текст встречи"))
		assert.Error(t, err)
	})
}

func TestChatService_GetAnswer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, conn := newEnv(t, http.StatusOK, `{"choices":[{"message":{"content":"ответ"}}]}`)
		svc := chatservice.New(srv.URL, conn)
		result, err := svc.GetAnswer(ctx, "вопрос", "контекст")
		require.NoError(t, err)
		assert.Equal(t, "ответ", result)
	})

	t.Run("api error", func(t *testing.T) {
		srv, conn := newEnv(t, http.StatusForbidden, ``)
		svc := chatservice.New(srv.URL, conn)
		_, err := svc.GetAnswer(ctx, "вопрос", "контекст")
		assert.Error(t, err)
	})
}

// TestChatService_TokenError проверяет путь, когда токен получить не удалось.
func TestChatService_TokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	conn := chat.New(srv.URL, "uid", "key")
	// Connect падает — состояние ошибки сохраняется в соединении.
	_ = conn.Connect(context.Background())

	svc := chatservice.New(srv.URL, conn)
	_, err := svc.GetShort(context.Background(), []byte("text"))
	assert.Error(t, err)

	_, err = svc.GetAnswer(context.Background(), "q", "c")
	assert.Error(t, err)
}
