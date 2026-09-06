package mock

import (
	"context"
	"sync"

	"github.com/Piktet/MeetScribe/internal/model"
)

// MockLLMClient — заглушка для LLMClient (GigaChat API).
// Позволяет настраивать поведение методов и проверять вызовы.
type MockLLMClient struct {
	mx sync.Mutex

	// Ожидаемые значения для GetShort
	GetShortResult string
	GetShortError  error

	// Ожидаемые значения для GetAnswer
	GetAnswerResult string
	GetAnswerError  error

	// Счётчики вызовов
	GetShortCalls  int
	GetAnswerCalls int

	// Записанные аргументы
	GetShortInput   []byte
	GetAnswerPrompt string
	GetAnswerCtx    string
}

// Compile-time check: проверяем, что MockLLMClient реализует model.LLMClient.
var _ model.LLMClient = (*MockLLMClient)(nil)

// GetShort — заглушка метода GetShort.
// Возвращает заданную выжимку и ошибку. Записывает переданный текст.
func (m *MockLLMClient) GetShort(ctx context.Context, text []byte) (string, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.GetShortCalls++
	m.GetShortInput = text

	return m.GetShortResult, m.GetShortError
}

// GetAnswer — заглушка метода GetAnswer.
// Возвращает заданный ответ и ошибку. Записывает переданные аргументы.
func (m *MockLLMClient) GetAnswer(ctx context.Context, prompt, context string) (string, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.GetAnswerCalls++
	m.GetAnswerPrompt = prompt
	m.GetAnswerCtx = context

	return m.GetAnswerResult, m.GetAnswerError
}

// Reset сбрасывает все счётчики и записанные данные.
func (m *MockLLMClient) Reset() {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.GetShortCalls = 0
	m.GetAnswerCalls = 0
	m.GetShortInput = nil
	m.GetAnswerPrompt = ""
	m.GetAnswerCtx = ""
}
