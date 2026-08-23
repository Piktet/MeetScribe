// Package model — доменные модели системы.
package model

import (
	"context"
	"database/sql"
	"io"
	"time"
)

// TranscriptionStatus — статус обработки транскрипции.
type TranscriptionStatus string

const (
	StatusPending    TranscriptionStatus = "PENDING"    // задача создана, ожидает обработки
	StatusUploading  TranscriptionStatus = "UPLOADING"  // файл загружается
	StatusProcessing TranscriptionStatus = "PROCESSING" // распознавание в процессе
	StatusDone       TranscriptionStatus = "DONE"       // успешно завершена
	StatusError      TranscriptionStatus = "ERROR"      // ошибка обработки
	StatusCanceled   TranscriptionStatus = "CANCELED"   // отменена
)

// Transcription — встреча/аудиофайл с результатами обработки.
type Transcription struct {
	ID            string // уникальный ID транскрипции
	UserID        int64  // ID пользователя Telegram
	ChatID        int64  // ID чата Telegram
	Name          string // имя/название встречи
	FilePath      string // ID файла в SaluteSpeech (входной)
	OutputFileID  string // ID файла с результатом
	TaskID        string // ID задачи в SaluteSpeech
	Transcription string // полная транскрипция
	Summary       string // краткая выжимка от LLM
	Status        TranscriptionStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ResultStatusType — статус задачи распознавания в SaluteSpeech.
type ResultStatusType string

const (
	SpeachResultStatusEmpty    ResultStatusType = ""
	SpeachResultStatusNew      ResultStatusType = "NEW"
	SpeachResultStatusRunning  ResultStatusType = "RUNNING"
	SpeachResultStatusDone     ResultStatusType = "DONE"
	SpeachResultStatusError    ResultStatusType = "ERROR"
	SpeachResultStatusCanceled ResultStatusType = "CANCELED"
)

// SpeachTaskData — данные для задачи распознавания.
type SpeachTaskData struct {
	User   int64     // ID пользователя
	ChatID int64     // ID чата
	Name   string    // название встречи/файла
	Input  io.Reader // поток аудио или текста
}

// SpeachTaskResponse — результат задачи распознавания.
type SpeachTaskResponse struct {
	*SpeachTaskData
	InFileID  string
	OutFileID string
	TaskID    string
	Output    []byte
}

// SpeechUploadResponse — ответ API на загрузку аудиофайла.
type SpeechUploadResponse struct {
	Status int `json:"status"`
	Result struct {
		FileID string `json:"request_file_id"`
	} `json:"result"`
}

// SpeachUploadResponse — ответ API на загрузку аудиофайла.
type SpeachUploadResponse struct {
	Status int `json:"status"`
	Result struct {
		FileID string `json:"request_file_id"`
	} `json:"result"`
}

// SpeechCreateTaskOptionRequest — параметры создания задачи распознавания.
type SpeechCreateTaskOptionRequest struct {
	AudioEncoding string `json:"audio_encoding"`
}

// SpeachCreateTaskOptionRequest — параметры создания задачи распознавания.
type SpeachCreateTaskOptionRequest struct {
	AudioEncoding string `json:"audio_encoding"`
}

// SpeechCreateTaskRequest — запрос на создание задачи распознавания речи.
type SpeechCreateTaskRequest struct {
	FileID  string                        `json:"request_file_id"`
	Options SpeechCreateTaskOptionRequest `json:"options"`
}

// SpeachCreateTaskRequest — запрос на создание задачи распознавания речи.
type SpeachCreateTaskRequest struct {
	FileID  string                        `json:"request_file_id"`
	Options SpeachCreateTaskOptionRequest `json:"options"`
}

// SpeechCreateTaskResultResponse — результат создания задачи.
type SpeechCreateTaskResultResponse struct {
	ID      string              `json:"id"`
	Created string              `json:"created_at"`
	Updated string              `json:"updated_at"`
	Status  TranscriptionStatus `json:"status"`
	FileID  string              `json:"response_file_id,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// SpeachCreateTaskResultResponse — результат создания задачи.
type SpeachCreateTaskResultResponse struct {
	ID      string           `json:"id"`
	Created string           `json:"created_at"`
	Updated string           `json:"updated_at"`
	Status  ResultStatusType `json:"status"`
	FileID  string           `json:"response_file_id,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// SpeechCreateTaskResponse — ответ API на создание задачи.
type SpeechCreateTaskResponse struct {
	Status int                            `json:"status"`
	Result SpeechCreateTaskResultResponse `json:"result"`
}

// SpeachCreateTaskResponse — ответ API на создание задачи.
type SpeachCreateTaskResponse struct {
	Status int                            `json:"status"`
	Result SpeachCreateTaskResultResponse `json:"result"`
}

// TranscriptionTaskData — данные для задачи распознавания.
type TranscriptionTaskData struct {
	UserID int64     // ID пользователя
	ChatID int64     // ID чата
	Name   string    // название встречи/файла
	Input  io.Reader // поток аудио или текста
}

// TranscriptionTaskResponse — результат задачи распознавания.
type TranscriptionTaskResponse struct {
	*TranscriptionTaskData
	TranscriptionID string
	InFileID        string
	OutFileID       string
	TaskID          string
	Output          []byte
}

// LLMChatRequest — запрос к LLM-клиенту.
type LLMChatRequest struct {
	UserID  int64
	ChatID  int64
	Prompt  string
	Context string // контекст: транскрипция или выжимка
}

// FileInfo — информация о файле.
type FileInfo struct {
	ID            string
	ChatID        string
	TaskID        string
	FilePath      string
	Transcription string
	Summary       string
	Status        TranscriptionStatus
	CreatedAt     time.Time
}

// Connection — интерфейс подключения к БД.
type Connection interface {
	// Query выполняет SELECT-запрос.
	Query(context.Context, string, ...any) (*sql.Rows, error)
	// Execute выполняет команду (INSERT, UPDATE, DELETE, CREATE).
	Execute(context.Context, string, ...any) error
	// BeginTx начинает новую транзакцию.
	BeginTx(context.Context) (*sql.Tx, error)
}

// SpeechClient — абстрактный клиент для распознавания речи.
type SpeechClient interface {
	// Upload загружает аудиофайл. Возвращает ID загруженного файла.
	Upload(ctx context.Context, data io.Reader) (string, error)
	// CreateTask создает задачу распознавания. Возвращает ID задачи, ID результата, статус.
	CreateTask(ctx context.Context, fileID string) (string, string, TranscriptionStatus, error)
	// GetStatus получает статус задачи. Возвращает ID результата, статус, флаг повтора.
	GetStatus(ctx context.Context, taskID string) (string, TranscriptionStatus, bool, error)
	// Download скачивает результат распознавания.
	Download(ctx context.Context, fileID string) ([]byte, error)
}

// LLMClient — абстрактный клиент для LLM (GigaChat и т.п.).
type LLMClient interface {
	// GetShort получает краткую выжимку из текста.
	GetShort(ctx context.Context, text []byte) (string, error)
	// GetAnswer отвечает на вопрос по контексту (транскрипция/выжимка).
	GetAnswer(ctx context.Context, prompt, context string) (string, error)
}
