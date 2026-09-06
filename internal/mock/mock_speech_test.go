package mock_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/Piktet/MeetScribe/internal/mock"
	"github.com/Piktet/MeetScribe/internal/model"
)

func TestMockSpeechClient_BasicUsage(t *testing.T) {
	m := mock.NewMockSpeechClient()

	// Настраиваем ожидаемое поведение через builders
	m.OnUpload().Return("file-123", nil)
	m.OnCreateTask().Return("task-1", "result-1", model.StatusDone, nil)
	m.OnGetStatus().Return("result-1", model.StatusDone, false, nil)
	m.OnDownload().Return([]byte("transcription"), nil)

	ctx := context.Background()

	// Test Upload
	fileID, err := m.Upload(ctx, strings.NewReader("audio data"))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if fileID != "file-123" {
		t.Errorf("expected file-123, got %s", fileID)
	}

	// Test CreateTask
	taskID, resultFileID, status, err := m.CreateTask(ctx, "file-123")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if taskID != "task-1" {
		t.Errorf("expected task-1, got %s", taskID)
	}
	if resultFileID != "result-1" {
		t.Errorf("expected result-1, got %s", resultFileID)
	}
	if status != model.StatusDone {
		t.Errorf("expected StatusDone, got %s", status)
	}

	// Test GetStatus
	resultFileID, status, retry, err := m.GetStatus(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if resultFileID != "result-1" {
		t.Errorf("expected result-1, got %s", resultFileID)
	}
	if status != model.StatusDone {
		t.Errorf("expected StatusDone, got %s", status)
	}
	if retry {
		t.Error("expected retry=false")
	}

	// Test Download
	data, err := m.Download(ctx, "result-1")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if string(data) != "transcription" {
		t.Errorf("expected transcription, got %s", string(data))
	}
}

func TestMockSpeechClient_WithFunctions(t *testing.T) {
	m := mock.NewMockSpeechClient()

	// Динамическое поведение через функции
	m.SetUploadFunc(func(ctx context.Context, data io.Reader) (string, error) {
		if data == nil {
			return "", nil
		}
		return "dynamic-file-id", nil
	})

	ctx := context.Background()
	fileID, err := m.Upload(ctx, strings.NewReader("test"))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if fileID != "dynamic-file-id" {
		t.Errorf("expected dynamic-file-id, got %s", fileID)
	}
}

func TestMockSpeechClient_AssertExpectations(t *testing.T) {
	m := mock.NewMockSpeechClient()

	// Задаём ожидания
	m.OnUpload().Return("file-1", nil)
	m.OnCreateTask().Return("task-1", "result-1", model.StatusDone, nil)

	ctx := context.Background()

	// Выполняем только Upload, но не CreateTask
	_, _ = m.Upload(ctx, strings.NewReader("data"))

	// Проверяем ожидания — должно быть сообщение о невыполненном CreateTask
	missing := m.AssertExpectations()
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing expectation, got %d: %v", len(missing), missing)
	}
	if missing[0] != "CreateTask не был вызван" {
		t.Errorf("unexpected message: %s", missing[0])
	}
}

func TestMockSpeechClient_CallsCount(t *testing.T) {
	m := mock.NewMockSpeechClient()

	m.OnUpload().Return("file-1", nil)
	m.OnCreateTask().Return("task-1", "result-1", model.StatusDone, nil)

	ctx := context.Background()

	// Вызываем несколько раз
	for i := 0; i < 3; i++ {
		_, _ = m.Upload(ctx, strings.NewReader("data"))
	}
	for i := 0; i < 2; i++ {
		_, _, _, _ = m.CreateTask(ctx, "file-1")
	}

	if m.UploadCallsCount() != 3 {
		t.Errorf("expected 3 Upload calls, got %d", m.UploadCallsCount())
	}
	if m.CreateTaskCallsCount() != 2 {
		t.Errorf("expected 2 CreateTask calls, got %d", m.CreateTaskCallsCount())
	}
}

func TestMockSpeechClient_ArumentRecording(t *testing.T) {
	m := mock.NewMockSpeechClient()

	m.OnUpload().Return("file-1", nil)
	m.OnCreateTask().Return("task-1", "result-1", model.StatusDone, nil)
	m.OnDownload().Return([]byte("result"), nil)

	ctx := context.Background()

	testData := []byte("audio-payload")
	_, _ = m.Upload(ctx, strings.NewReader(string(testData)))
	_, _, _, _ = m.CreateTask(ctx, "file-1")
	_, _ = m.Download(ctx, "result-1")

	// Проверяем записанные аргументы
	uploaded := m.LastUploadInput()
	if string(uploaded) != "audio-payload" {
		t.Errorf("expected 'audio-payload', got %q", string(uploaded))
	}

	if m.LastCreateTaskFileID() != "file-1" {
		t.Errorf("expected 'file-1', got %q", m.LastCreateTaskFileID())
	}

	if m.LastDownloadFileID() != "result-1" {
		t.Errorf("expected 'result-1', got %q", m.LastDownloadFileID())
	}
}

func TestMockSpeechClient_Reset(t *testing.T) {
	m := mock.NewMockSpeechClient()

	m.OnUpload().Return("file-1", nil)
	m.OnCreateTask().Return("task-1", "result-1", model.StatusDone, nil)

	ctx := context.Background()
	_, _ = m.Upload(ctx, strings.NewReader("data"))
	_, _, _, _ = m.CreateTask(ctx, "file-1")

	if m.UploadCallsCount() != 1 {
		t.Errorf("expected 1 call before reset, got %d", m.UploadCallsCount())
	}

	m.Reset()

	if m.UploadCallsCount() != 0 {
		t.Errorf("expected 0 calls after reset, got %d", m.UploadCallsCount())
	}

	// После Reset ожидания и счётчики должны быть сброшены
	// AssertExpectations должен вернуть 0, так как expect* = false
	missing := m.AssertExpectations()
	if len(missing) != 0 {
		t.Errorf("expected 0 missing expectations after reset, got %d: %v", len(missing), missing)
	}
}

func TestMockSpeechClient_ErrorHandling(t *testing.T) {
	m := mock.NewMockSpeechClient()

	m.OnUpload().Return("", nil)
	m.OnCreateTask().Return("task-1", "result-1", model.StatusDone, nil)
	m.OnGetStatus().Return("", model.StatusError, false, nil)
	m.OnDownload().Return(nil, nil)

	ctx := context.Background()

	// Upload без ошибки
	fileID, err := m.Upload(ctx, strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Upload should not error, got: %v", err)
	}
	if fileID != "" {
		t.Errorf("expected empty fileID, got %q", fileID)
	}

	// GetStatus с ошибкой
	_, status, _, err := m.GetStatus(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetStatus should not error, got: %v", err)
	}
	if status != model.StatusError {
		t.Errorf("expected StatusError, got %s", status)
	}
}
