// Package mock предоставляет заглушки для внешних API, используемых в тестировании.
package mock

import (
	"context"
	"io"
	"sync"

	"github.com/Piktet/MeetScribe/internal/model"
)

// MockSpeechClient — заглушка для SpeechClient (SaluteSpeech API).
// Позволяет настраивать поведение методов и проверять вызовы.
//
// Пример использования:
//
//	mock := mock.NewMockSpeechClient()
//	mock.OnUpload().Return("file-id-123", nil)
//	mock.OnCreateTask().Return("task-1", "result-1", model.StatusDone, nil)
//
//	// ... использование mock ...
//
//	mock.AssertExpectations(t) // проверка всех ожиданий
type MockSpeechClient struct {
	mx sync.Mutex

	// Функции для динамического поведения
	uploadFn     func(ctx context.Context, data io.Reader) (string, error)
	createTaskFn func(ctx context.Context, fileID string) (string, string, model.TranscriptionStatus, error)
	getStatusFn  func(ctx context.Context, taskID string) (string, model.TranscriptionStatus, bool, error)
	downloadFn   func(ctx context.Context, fileID string) ([]byte, error)

	// Ожидаемые значения по умолчанию (если функции не заданы)
	uploadFileID     string
	uploadError      error
	createTaskTaskID string
	createTaskResult string
	createTaskStatus model.TranscriptionStatus
	createTaskError  error
	getStatusResult  string
	getStatusStatus  model.TranscriptionStatus
	getStatusRetry   bool
	getStatusError   error
	downloadData     []byte
	downloadError    error

	// Ожидания (задаются через On...)
	expectUpload     bool
	expectCreateTask bool
	expectGetStatus  bool
	expectDownload   bool

	// Счётчики вызовов
	uploadCalls     int
	createTaskCalls int
	getStatusCalls  int
	downloadCalls   int

	// Записанные аргументы
	uploadInput            []byte
	createTaskFileIDActual string
	downloadFileIDActual   string
}

// Compile-time check.
var _ model.SpeechClient = (*MockSpeechClient)(nil)

// NewMockSpeechClient создаёт новый экземпляр MockSpeechClient.
func NewMockSpeechClient() *MockSpeechClient {
	return &MockSpeechClient{}
}

// --- Настройка ожиданий ---

// OnUpload задаёт ожидание вызова Upload.
func (m *MockSpeechClient) OnUpload() *UploadExpectation {
	m.expectUpload = true
	return &UploadExpectation{mock: m}
}

// UploadExpectation — builder для настройки Upload.
type UploadExpectation struct {
	mock *MockSpeechClient
}

// Return задаёт возвращаемые значения для Upload.
func (e *UploadExpectation) Return(fileID string, err error) {
	e.mock.uploadFileID = fileID
	e.mock.uploadError = err
}

// OnCreateTask задаёт ожидание вызова CreateTask.
func (m *MockSpeechClient) OnCreateTask() *CreateTaskExpectation {
	m.expectCreateTask = true
	return &CreateTaskExpectation{mock: m}
}

// CreateTaskExpectation — builder для настройки CreateTask.
type CreateTaskExpectation struct {
	mock *MockSpeechClient
}

// Return задаёт возвращаемые значения для CreateTask.
func (e *CreateTaskExpectation) Return(taskID, resultFileID string, status model.TranscriptionStatus, err error) {
	e.mock.createTaskTaskID = taskID
	e.mock.createTaskResult = resultFileID
	e.mock.createTaskStatus = status
	e.mock.createTaskError = err
}

// OnGetStatus задаёт ожидание вызова GetStatus.
func (m *MockSpeechClient) OnGetStatus() *GetStatusExpectation {
	m.expectGetStatus = true
	return &GetStatusExpectation{mock: m}
}

// GetStatusExpectation — builder для настройки GetStatus.
type GetStatusExpectation struct {
	mock *MockSpeechClient
}

// Return задаёт возвращаемые значения для GetStatus.
func (e *GetStatusExpectation) Return(resultFileID string, status model.TranscriptionStatus, retry bool, err error) {
	e.mock.getStatusResult = resultFileID
	e.mock.getStatusStatus = status
	e.mock.getStatusRetry = retry
	e.mock.getStatusError = err
}

// OnDownload задаёт ожидание вызова Download.
func (m *MockSpeechClient) OnDownload() *DownloadExpectation {
	m.expectDownload = true
	return &DownloadExpectation{mock: m}
}

// DownloadExpectation — builder для настройки Download.
type DownloadExpectation struct {
	mock *MockSpeechClient
}

// Return задаёт возвращаемые значения для Download.
func (e *DownloadExpectation) Return(data []byte, err error) {
	e.mock.downloadData = data
	e.mock.downloadError = err
}

// --- Динамическое поведение ---

// SetUploadFunc задаёт функцию для Upload.
func (m *MockSpeechClient) SetUploadFunc(fn func(ctx context.Context, data io.Reader) (string, error)) {
	m.uploadFn = fn
}

// SetCreateTaskFunc задаёт функцию для CreateTask.
func (m *MockSpeechClient) SetCreateTaskFunc(fn func(ctx context.Context, fileID string) (string, string, model.TranscriptionStatus, error)) {
	m.createTaskFn = fn
}

// SetGetStatusFunc задаёт функцию для GetStatus.
func (m *MockSpeechClient) SetGetStatusFunc(fn func(ctx context.Context, taskID string) (string, model.TranscriptionStatus, bool, error)) {
	m.getStatusFn = fn
}

// SetDownloadFunc задаёт функцию для Download.
func (m *MockSpeechClient) SetDownloadFunc(fn func(ctx context.Context, fileID string) ([]byte, error)) {
	m.downloadFn = fn
}

// --- Реализация интерфейса SpeechClient ---

// Upload — заглушка метода Upload.
func (m *MockSpeechClient) Upload(ctx context.Context, data io.Reader) (string, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.uploadCalls++

	if m.uploadFn != nil {
		return m.uploadFn(ctx, data)
	}

	if data != nil {
		b, err := io.ReadAll(data)
		if err != nil {
			m.uploadError = err
			return "", err
		}
		m.uploadInput = b
	}

	return m.uploadFileID, m.uploadError
}

// CreateTask — заглушка метода CreateTask.
func (m *MockSpeechClient) CreateTask(ctx context.Context, fileID string) (string, string, model.TranscriptionStatus, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.createTaskCalls++
	m.createTaskFileIDActual = fileID

	if m.createTaskFn != nil {
		return m.createTaskFn(ctx, fileID)
	}

	return m.createTaskTaskID, m.createTaskResult, m.createTaskStatus, m.createTaskError
}

// GetStatus — заглушка метода GetStatus.
func (m *MockSpeechClient) GetStatus(ctx context.Context, taskID string) (string, model.TranscriptionStatus, bool, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.getStatusCalls++

	if m.getStatusFn != nil {
		return m.getStatusFn(ctx, taskID)
	}

	return m.getStatusResult, m.getStatusStatus, m.getStatusRetry, m.getStatusError
}

// Download — заглушка метода Download.
func (m *MockSpeechClient) Download(ctx context.Context, fileID string) ([]byte, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.downloadCalls++
	m.downloadFileIDActual = fileID

	if m.downloadFn != nil {
		return m.downloadFn(ctx, fileID)
	}

	return m.downloadData, m.downloadError
}

// --- Проверка ожиданий ---

// AssertExpectations проверяет, что все заданные ожидания были выполнены.
// Возвращает список невыполненных ожиданий.
func (m *MockSpeechClient) AssertExpectations() []string {
	m.mx.Lock()
	defer m.mx.Unlock()

	var missing []string

	if m.expectUpload && m.uploadCalls == 0 {
		missing = append(missing, "Upload не был вызван")
	}
	if m.expectCreateTask && m.createTaskCalls == 0 {
		missing = append(missing, "CreateTask не был вызван")
	}
	if m.expectGetStatus && m.getStatusCalls == 0 {
		missing = append(missing, "GetStatus не был вызван")
	}
	if m.expectDownload && m.downloadCalls == 0 {
		missing = append(missing, "Download не был вызван")
	}

	return missing
}

// --- Информация о вызовах ---

// UploadCallsCount возвращает количество вызовов Upload.
func (m *MockSpeechClient) UploadCallsCount() int {
	m.mx.Lock()
	defer m.mx.Unlock()
	return m.uploadCalls
}

// CreateTaskCallsCount возвращает количество вызовов CreateTask.
func (m *MockSpeechClient) CreateTaskCallsCount() int {
	m.mx.Lock()
	defer m.mx.Unlock()
	return m.createTaskCalls
}

// GetStatusCallsCount возвращает количество вызовов GetStatus.
func (m *MockSpeechClient) GetStatusCallsCount() int {
	m.mx.Lock()
	defer m.mx.Unlock()
	return m.getStatusCalls
}

// DownloadCallsCount возвращает количество вызовов Download.
func (m *MockSpeechClient) DownloadCallsCount() int {
	m.mx.Lock()
	defer m.mx.Unlock()
	return m.downloadCalls
}

// LastUploadInput возвращает данные последнего вызова Upload.
func (m *MockSpeechClient) LastUploadInput() []byte {
	m.mx.Lock()
	defer m.mx.Unlock()
	result := make([]byte, len(m.uploadInput))
	copy(result, m.uploadInput)
	return result
}

// LastCreateTaskFileID возвращает fileID последнего вызова CreateTask.
func (m *MockSpeechClient) LastCreateTaskFileID() string {
	m.mx.Lock()
	defer m.mx.Unlock()
	return m.createTaskFileIDActual
}

// LastDownloadFileID возвращает fileID последнего вызова Download.
func (m *MockSpeechClient) LastDownloadFileID() string {
	m.mx.Lock()
	defer m.mx.Unlock()
	return m.downloadFileIDActual
}

// Reset сбрасывает все счётчики, данные и ожидания.
func (m *MockSpeechClient) Reset() {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.uploadCalls = 0
	m.createTaskCalls = 0
	m.getStatusCalls = 0
	m.downloadCalls = 0
	m.uploadInput = nil
	m.createTaskFileIDActual = ""
	m.downloadFileIDActual = ""

	m.expectUpload = false
	m.expectCreateTask = false
	m.expectGetStatus = false
	m.expectDownload = false

	m.uploadFn = nil
	m.createTaskFn = nil
	m.getStatusFn = nil
	m.downloadFn = nil
}
