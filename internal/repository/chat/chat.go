// Package chat — взаимодействие с GigaChat API.
package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"unsafe"

	"github.com/Piktet/tg_bot/internal/logger"

	"go.uber.org/zap"
)

// Message — сообщение в чате (роль и контент).
type Message struct {
	Role    string `json:"role"`    // роль: "user" или "assistant"
	Content string `json:"content"` // текст сообщения
}

// ChatRequest — запрос к GigaChat API.
type ChatRequest struct {
	Model    string     `json:"model"`    // модель: "GigaChat"
	Messages []*Message `json:"messages"` // список сообщений
}

// ChatResponse — ответ от GigaChat API.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"` // сгенерированный контент
		} `json:"message"`
	} `json:"choices"`
}

const (
	// promptShort — промпт для получения краткой выжимки.
	promptShort = "сделай краткую выжимку из текста %s"
	// promptAnswer — промпт для ответа на вопрос по контексту.
	promptAnswer = "контекст:\n%s\n\nвопрос: %s"
)

// GetShort — получить краткую выжимку из текста через GigaChat.
// host — хост API, token — токен авторизации, text — исходный текст.
func GetShort(ctx context.Context, host, token string, text []byte) (string, error) {
	return getData(ctx, host, token, "user", fmt.Sprintf(promptShort, unsafe.String(unsafe.SliceData(text), len(text))))
}

// GetAnswer — получить ответ на вопрос по контексту через GigaChat.
// host — хост API, token — токен авторизации, prompt — вопрос, context — контекст (транскрипция/выжимка).
func GetAnswer(ctx context.Context, host, token, question, context string) (string, error) {
	return getData(ctx, host, token, "user", fmt.Sprintf(promptAnswer, context, question))
}

// getData — отправить запрос к GigaChat API и вернуть ответ.
// role — роль отправителя сообщения, content — текст запроса.
func getData(ctx context.Context, host, token, role string, content string) (string, error) {
	u, err := url.JoinPath(host, "/api/v1/chat/completions")
	if err != nil {
		logger.Log().Error("getData - create path error", zap.Error(err))
		return "", err
	}

	data := &ChatRequest{
		Model: "GigaChat",
		Messages: []*Message{{
			Role:    role,
			Content: content,
		}},
	}

	body, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	r, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewBuffer(body))
	if err != nil {
		logger.Log().Error("UploadVoice - create request", zap.Error(err))
		return "", err
	}

	r.Header.Add("Authorization", "Bearer "+token)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		logger.Log().Error("UploadVoice - send request", zap.Error(err))
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := errors.New("ошибка авторизации")
		logger.Log().Error("UploadVoice - get response", zap.Error(err))
		return "", err
	}

	var x ChatResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&x); err != nil {
		logger.Log().Error("UploadVoice - decode result", zap.Error(err))
		return "", err
	}

	if len(x.Choices) > 0 {
		return x.Choices[0].Message.Content, nil
	}

	return "", nil

}
