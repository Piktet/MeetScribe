package speach

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Piktet/MeetScribe/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer создает тестовый HTTP-сервер с заданным обработчиком.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestUpload(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
			assert.Equal(t, "audio/mpeg", r.Header.Get("Content-Type"))
			w.Write([]byte(`{"status":200,"result":{"request_file_id":"file-1"}}`))
		})
		fileID, err := Upload(ctx, srv.URL, "tok", strings.NewReader("audio"))
		require.NoError(t, err)
		assert.Equal(t, "file-1", fileID)
	})

	t.Run("http error", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		_, err := Upload(ctx, srv.URL, "tok", strings.NewReader("audio"))
		assert.ErrorContains(t, err, "Internal Server Error")
	})

	t.Run("api status error", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"status":400,"result":{"request_file_id":""}}`))
		})
		_, err := Upload(ctx, srv.URL, "tok", strings.NewReader("audio"))
		assert.ErrorContains(t, err, "Bad Request")
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`not json`))
		})
		_, err := Upload(ctx, srv.URL, "tok", strings.NewReader("audio"))
		assert.Error(t, err)
	})

	t.Run("unreachable host", func(t *testing.T) {
		_, err := Upload(ctx, "http://127.0.0.1:1", "tok", strings.NewReader("audio"))
		assert.Error(t, err)
	})
}

func TestCreateTask(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			w.Write([]byte(`{"status":200,"result":{"id":"task-1","response_file_id":"out-1","status":"NEW"}}`))
		})
		taskID, fileID, status, err := CreateTask(ctx, srv.URL, "tok", "file-1")
		require.NoError(t, err)
		assert.Equal(t, "task-1", taskID)
		assert.Equal(t, "out-1", fileID)
		assert.Equal(t, model.SpeachResultStatusNew, status)
	})

	t.Run("http error", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		_, _, _, err := CreateTask(ctx, srv.URL, "tok", "file-1")
		assert.ErrorContains(t, err, "Forbidden")
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{`))
		})
		_, _, _, err := CreateTask(ctx, srv.URL, "tok", "file-1")
		assert.Error(t, err)
	})
}

func TestGetStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("success new", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"status":200,"result":{"id":"task-1","status":"NEW"}}`))
		})
		fileID, status, retry, err := GetStatus(ctx, srv.URL, "tok", "task-1")
		require.NoError(t, err)
		assert.False(t, retry)
		assert.Equal(t, model.SpeachResultStatusNew, status)
		assert.Empty(t, fileID)
	})

	t.Run("success done", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"status":200,"result":{"id":"task-1","response_file_id":"out-1","status":"DONE"}}`))
		})
		fileID, status, retry, err := GetStatus(ctx, srv.URL, "tok", "task-1")
		require.NoError(t, err)
		assert.False(t, retry)
		assert.Equal(t, model.SpeachResultStatusDone, status)
		assert.Equal(t, "out-1", fileID)
	})

	t.Run("internal server error is retryable", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		_, _, retry, err := GetStatus(ctx, srv.URL, "tok", "task-1")
		assert.Error(t, err)
		assert.True(t, retry)
	})

	t.Run("other http error is not retryable", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		_, _, retry, err := GetStatus(ctx, srv.URL, "tok", "task-1")
		assert.Error(t, err)
		assert.False(t, retry)
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`invalid`))
		})
		_, _, _, err := GetStatus(ctx, srv.URL, "tok", "task-1")
		assert.Error(t, err)
	})
}

func TestDownload(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Query().Get("response_file_id"), "out-1")
			w.Write([]byte("transcription text"))
		})
		data, err := Download(ctx, srv.URL, "tok", "out-1")
		require.NoError(t, err)
		assert.Equal(t, "transcription text", string(data))
	})

	t.Run("http error", func(t *testing.T) {
		srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})
		_, err := Download(ctx, srv.URL, "tok", "out-1")
		assert.Error(t, err)
	})

	t.Run("unreachable host", func(t *testing.T) {
		_, err := Download(ctx, "http://127.0.0.1:1", "tok", "out-1")
		assert.Error(t, err)
	})
}

func TestParseStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		id, fileID, status, err := parseStatus(strings.NewReader(
			`{"status":200,"result":{"id":"task-1","response_file_id":"out-1","status":"RUNNING"}}`))
		require.NoError(t, err)
		assert.Equal(t, "task-1", id)
		assert.Equal(t, "out-1", fileID)
		assert.Equal(t, model.SpeachResultStatusRunning, status)
	})

	t.Run("api status error", func(t *testing.T) {
		_, _, _, err := parseStatus(strings.NewReader(`{"status":401,"result":{}}`))
		assert.ErrorContains(t, err, "Unauthorized")
	})

	t.Run("invalid json", func(t *testing.T) {
		_, _, _, err := parseStatus(strings.NewReader(`[[[`))
		assert.Error(t, err)
	})
}

// TestParseStatusUnknown проверяет парсинг неизвестного статуса.
func TestParseStatusUnknown(t *testing.T) {
	_, _, status, err := parseStatus(strings.NewReader(
		`{"status":200,"result":{"id":"task-1","status":"WEIRD"}}`))
	require.NoError(t, err)
	assert.Equal(t, model.ResultStatusType("WEIRD"), status)
}

// TestUploadReaderError проверяет обработку ошибки чтения входного потока.
func TestUploadReaderError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":200,"result":{"request_file_id":"file-1"}}`))
	})
	_, err := Upload(context.Background(), srv.URL, "tok", errReader{})
	assert.Error(t, err)
}

// errReader всегда возвращает ошибку при чтении.
type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, errors.New("read error") }
