package speach

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAuthServer создает сервер, эмулирующий эндпоинт авторизации SaluteSpeech.
func newAuthServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/oauth", r.URL.Path)
		assert.Equal(t, "uid-1", r.Header.Get("RqUID"))
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSpeachConnection_Connect(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv := newAuthServer(t, http.StatusOK, `{"access_token":"tok-1","expires_at":4102444800}`)
		conn := New(srv.URL, "uid-1", "authkey")
		require.NoError(t, conn.Connect(ctx))
		token, err := conn.GetToken()
		require.NoError(t, err)
		assert.Equal(t, "tok-1", token)
	})

	t.Run("auth error", func(t *testing.T) {
		srv := newAuthServer(t, http.StatusUnauthorized, `unauthorized`)
		conn := New(srv.URL, "uid-1", "authkey")
		err := conn.Connect(ctx)
		assert.Error(t, err)
		// Ошибка сохраняется и возвращается из GetToken.
		_, err = conn.GetToken()
		assert.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := newAuthServer(t, http.StatusOK, `not-json`)
		conn := New(srv.URL, "uid-1", "authkey")
		err := conn.Connect(ctx)
		assert.Error(t, err)
	})

	t.Run("unreachable host", func(t *testing.T) {
		conn := New("http://127.0.0.1:1", "uid-1", "authkey")
		err := conn.Connect(ctx)
		assert.Error(t, err)
		_, err = conn.GetToken()
		assert.Error(t, err)
	})
}

func TestSpeachConnection_GetTokenWithoutConnect(t *testing.T) {
	conn := New("host", "uid", "key")
	token, err := conn.GetToken()
	require.NoError(t, err)
	assert.Empty(t, token)
}

// TestSpeachConnection_Reconnect проверяет, что после истечения токена
// запланировано переподключение (expires_at в прошлом + небольшой запас).
func TestSpeachConnection_Reconnect(t *testing.T) {
	// Атомарный счётчик: инкрементируется в горутине HTTP-сервера.
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// Токен истекает через 50 мс — AfterFunc должен переподключиться.
		w.Write([]byte(`{"access_token":"tok","expires_at":` +
			strconv.FormatInt(time.Now().Add(50*time.Millisecond).Unix(), 10) + `}`))
	}))
	defer srv.Close()

	conn := New(srv.URL, "uid", "key")
	require.NoError(t, conn.Connect(context.Background()))

	// Ждём больше, чем время жизни токена.
	time.Sleep(200 * time.Millisecond)
	assert.Greater(t, calls.Load(), int64(1), "expected reconnect attempt")
}
