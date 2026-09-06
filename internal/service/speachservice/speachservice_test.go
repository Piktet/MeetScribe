package speachservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Piktet/MeetScribe/internal/model"
	"github.com/Piktet/MeetScribe/internal/repository/speach"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpeachOption(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		o := newSpeachOption()
		assert.Equal(t, defaultStatusTimeout, o.statusTimeout)
		assert.Nil(t, o.connSpeach)
		assert.Empty(t, o.host)
	})

	t.Run("with options", func(t *testing.T) {
		conn := speach.New("host", "uid", "key")
		o := newSpeachOption(
			WithSpeach(conn),
			WithHost("example.com"),
			WithStatusTimeout(5*time.Second),
		)
		assert.Same(t, conn, o.connSpeach)
		assert.Equal(t, "example.com", o.host)
		assert.Equal(t, 5*time.Second, o.statusTimeout)
	})
}

func TestCheckSpeachStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    model.ResultStatusType
		wantRetry bool
		wantErr   bool
	}{
		{name: "done", status: model.SpeachResultStatusDone, wantRetry: false, wantErr: false},
		{name: "new", status: model.SpeachResultStatusNew, wantRetry: true, wantErr: false},
		{name: "running", status: model.SpeachResultStatusRunning, wantRetry: true, wantErr: false},
		{name: "canceled", status: model.SpeachResultStatusCanceled, wantRetry: false, wantErr: true},
		{name: "error", status: model.SpeachResultStatusError, wantRetry: false, wantErr: true},
		{name: "empty", status: model.SpeachResultStatusEmpty, wantRetry: false, wantErr: true},
		{name: "unknown", status: model.ResultStatusType("WEIRD"), wantRetry: false, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retry, err := checkSpeachStatus(tt.status)
			assert.Equal(t, tt.wantRetry, retry)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	svc := New(WithHost("example.com"))
	require.NotNil(t, svc)
	assert.Equal(t, "example.com", svc.host)
	assert.Equal(t, defaultStatusTimeout, svc.statusTimeout)
}

func TestStartCanceledContext(t *testing.T) {
	svc := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.Start(ctx, 2, 4)
	assert.ErrorIs(t, err, context.Canceled)
}

// newSpeechEnv создает тестовый сервер, эмулирующий SaluteSpeech API:
// авторизацию, загрузку, создание задачи, опрос статуса и скачивание.
func newSpeechEnv(t *testing.T, uploadStatus int, createStatus int) (*httptest.Server, *speach.SpeachConnection) {
	t.Helper()
	var statusCalls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/api/v2/oauth"):
			w.Write([]byte(`{"access_token":"tok-1","expires_at":4102444800}`))
		case strings.HasSuffix(r.URL.Path, "/data:upload"):
			w.WriteHeader(uploadStatus)
			w.Write([]byte(`{"status":200,"result":{"request_file_id":"in-file-1"}}`))
		case strings.HasSuffix(r.URL.Path, "async_recognize"):
			w.WriteHeader(createStatus)
			w.Write([]byte(`{"status":200,"result":{"id":"task-1","status":"NEW"}}`))
		case strings.Contains(r.URL.Path, "task:get"):
			if atomic.AddInt64(&statusCalls, 1) == 1 {
				w.Write([]byte(`{"status":200,"result":{"id":"task-1","status":"RUNNING"}}`))
				return
			}
			w.Write([]byte(`{"status":200,"result":{"id":"task-1","response_file_id":"out-file-1","status":"DONE"}}`))
		case strings.HasSuffix(r.URL.Path, "data:download"):
			w.Write([]byte("transcription result"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	conn := speach.New(srv.URL, "uid", "key")
	require.NoError(t, conn.Connect(context.Background()))
	return srv, conn
}

func TestTaskProcessSuccess(t *testing.T) {
	srv, conn := newSpeechEnv(t, http.StatusOK, http.StatusOK)

	task := NewTask(newSpeachOption(
		WithSpeach(conn),
		WithHost(srv.URL),
		WithStatusTimeout(10*time.Millisecond),
	))

	resp, err := task.Process(context.Background(), &model.SpeachTaskData{
		User:   1,
		ChatID: 2,
		Name:   "meeting",
		Input:  strings.NewReader("audio data"),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "in-file-1", resp.InFileID)
	assert.Equal(t, "task-1", resp.TaskID)
	assert.Equal(t, int64(1), resp.User)
	assert.Equal(t, "transcription result", string(resp.Output))
}

func TestTaskProcessUploadError(t *testing.T) {
	srv, conn := newSpeechEnv(t, http.StatusInternalServerError, http.StatusOK)

	task := NewTask(newSpeachOption(
		WithSpeach(conn),
		WithHost(srv.URL),
	))
	_, err := task.Process(context.Background(), &model.SpeachTaskData{
		Input: strings.NewReader("audio data"),
	})
	assert.Error(t, err)
}

func TestTaskProcessCreateTaskError(t *testing.T) {
	srv, conn := newSpeechEnv(t, http.StatusOK, http.StatusInternalServerError)

	task := NewTask(newSpeachOption(
		WithSpeach(conn),
		WithHost(srv.URL),
	))
	_, err := task.Process(context.Background(), &model.SpeachTaskData{
		Input: strings.NewReader("audio data"),
	})
	assert.Error(t, err)
}

func TestTaskProcessTokenError(t *testing.T) {
	// Соединение без Connect: хост недоступен, загрузка завершится ошибкой.
	conn := speach.New("http://127.0.0.1:1", "uid", "key")
	task := NewTask(newSpeachOption(
		WithSpeach(conn),
		WithHost("http://127.0.0.1:1"),
	))
	_, err := task.Process(context.Background(), &model.SpeachTaskData{
		Input: strings.NewReader("audio data"),
	})
	assert.Error(t, err)
}

// TestWorkerProcessesTask проверяет, что воркер забирает задачу из очереди
// и кладёт результат в канал ответов.
func TestWorkerProcessesTask(t *testing.T) {
	srv, conn := newSpeechEnv(t, http.StatusOK, http.StatusOK)

	svc := New(
		WithSpeach(conn),
		WithHost(srv.URL),
		WithStatusTimeout(10*time.Millisecond),
	)
	svc.chTaskRequest = make(chan *model.SpeachTaskData, 4)
	svc.chTaskResponse = make(chan *model.SpeachTaskResponse, 4)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- svc.Worker(ctx) }()

	svc.AddTask(&model.SpeachTaskData{
		User:   1,
		ChatID: 2,
		Name:   "meeting",
		Input:  strings.NewReader("audio data"),
	})

	select {
	case resp := <-svc.GetTaskResponse():
		require.NotNil(t, resp)
		assert.Equal(t, "task-1", resp.TaskID)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for task response")
	}

	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for worker to stop")
	}
}

// TestGetTaskResponse возвращает канал ответов.
func TestGetTaskResponse(t *testing.T) {
	svc := New()
	svc.chTaskResponse = make(chan *model.SpeachTaskResponse, 1)
	assert.NotNil(t, svc.GetTaskResponse())
}
