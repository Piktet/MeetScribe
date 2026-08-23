// Package chatservice — сервис для работы с GigaChat API.
package chatservice

import (
	"context"
	"sync"

	"github.com/Piktet/tg_bot/internal/repository/chat"
)

// ChatService — сервис для работы с GigaChat API.
// Управляет подключением и обеспечивает потокобезопасный доступ к API.
type ChatService struct {
	mx   sync.Mutex           // мьютекс для потокобезопасности
	host string               // хост API
	conn *chat.ChatConnection // подключение к GigaChat API
}

// New создает новый экземпляр ChatService.
// host — хост API, conn — подключение к GigaChat API.
func New(host string, conn *chat.ChatConnection) *ChatService {
	return &ChatService{
		host: host,
		conn: conn,
	}
}

// GetShort получает краткую выжимку из текста через GigaChat.
func (p *ChatService) GetShort(ctx context.Context, full []byte) (string, error) {
	p.mx.Lock()
	defer p.mx.Unlock()
	token, err := p.conn.GetToken()
	if err != nil {
		return "", err
	}

	return chat.GetShort(ctx, p.host, token, full)

}

// GetAnswer получает ответ от GigaChat на вопрос по контексту.
func (p *ChatService) GetAnswer(ctx context.Context, question, context string) (string, error) {
	p.mx.Lock()
	defer p.mx.Unlock()
	token, err := p.conn.GetToken()
	if err != nil {
		return "", err
	}

	return chat.GetAnswer(ctx, p.host, token, question, context)

}
