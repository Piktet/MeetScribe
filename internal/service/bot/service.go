// Package bot — сервис Telegram-бота.
package bot

import (
	"context"
	"time"

	"github.com/Piktet/tg_bot/internal/model"
	"github.com/Piktet/tg_bot/internal/repository/db"

	tele "gopkg.in/telebot.v3"
)

// SpeachTask — интерфейс для обработки задач распознавания речи.
type SpeachTask interface {
	// AddTask добавляет задачу на распознавание.
	AddTask(*model.SpeachTaskData)
	// GetTaskResponse возвращает канал для получения результатов задач.
	GetTaskResponse() chan *model.SpeachTaskResponse
}

// chatTask — интерфейс для работы с GigaChat API.
type chatTask interface {
	// GetShort получает краткую выжимку из текста.
	GetShort(context.Context, []byte) (string, error)
	// GetAnswer получает ответ на вопрос по контексту.
	GetAnswer(context.Context, string, string) (string, error)
}

// DBTask — интерфейс для работы с базой данных (пользователи).
type DBTask interface {
	// AddUser добавляет пользователя в базу данных.
	AddUser(context.Context, int64, int64, string) error
}

// Bot — основной сервис Telegram-бота.
// Управляет обработкой команд, взаимодействует с Speech и Chat API.
type Bot struct {
	b    *tele.Bot  // экземпляр Telegram-бота
	user *tele.User // текущий пользователь

	speachTaskProcessor SpeachTask       // процессор задач распознавания речи
	chatProcessor       chatTask         // процессор чата (GigaChat)
	conn                model.Connection // подключение к БД

	chCommand chan *Command // очередь команд
}

// New создает новый экземпляр бота и регистрирует обработчики команд.
// token — токен Telegram-бота, conn — подключение к БД,
// s — процессор задач распознавания речи, c — процессор чата.
func New(token string, conn model.Connection, s SpeachTask, c chatTask) *Bot {
	b, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil
	}
	x := &Bot{
		b:                   b,
		speachTaskProcessor: s,
		chatProcessor:       c,
		conn:                conn,
	}

	b.Handle("/start", x.HandlerStart)
	b.Handle("/list", x.HandlerList)
	b.Handle("/get", x.HandlerGet)
	b.Handle("/find", x.HandlerFind)
	b.Handle("/chat", x.Handlerchat)

	b.Handle(tele.OnAudio, x.HandlerOnAudio)
	b.Handle(tele.OnVoice, x.HandlerOnVoice)
	b.Handle(tele.OnText, x.HandlerOnText)

	return x
}

// getTaskResultProcess — обработчик результатов задач распознавания.
// Получает результаты из канала, генерирует краткую выжимку через GigaChat,
// и сохраняет результат в базу данных.
func (p *Bot) getTaskResultProcess(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case task := <-p.speachTaskProcessor.GetTaskResponse():
			if short, err := p.chatProcessor.GetShort(ctx, task.Output); err == nil {
				db.AddTask(ctx, p.conn, task, short)
			}
		}
	}
}
