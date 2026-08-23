// Package bot — Telegram-бот: обработка команд и взаимодействие с внешними API.
package bot

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/Piktet/tg_bot/internal/logger"
	"github.com/Piktet/tg_bot/internal/model"
	"github.com/Piktet/tg_bot/internal/repository/db"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	tele "gopkg.in/telebot.v3"
)

// Command — команда для обработки ботом.
type Command struct {
	Name string                      // имя команды
	fn   func(context.Context) error // функция-обработчик команды
}

// Start запускает бота и воркеры обработки команд.
// cnt — количество воркеров, size — размер очереди команд.
func (p *Bot) Start(ctx context.Context, cnt, size int) error {
	p.chCommand = make(chan *Command, size)
	defer close(p.chCommand)
	var wg errgroup.Group
	for range cnt {
		wg.Go(func() error {
			return p.worker(ctx)
		})
	}
	wg.Go(func() error {
		go p.b.Start()
		<-ctx.Done()
		p.b.Stop()
		return nil
	})

	wg.Go(func() error {
		return p.getTaskResultProcess(ctx)
	})
	return wg.Wait()

}

// addCommand добавляет команду в очередь обработки.
// Если очередь заполнена, отправляет пользователю сообщение "try later".
func (p *Bot) addCommand(c tele.Context, cmd *Command) error {
	select {
	case p.chCommand <- cmd:
		logger.Log().Info("command added", zap.String("name", cmd.Name))
		return nil
	default:
		return c.Send("try later")
	}
}

// worker — воркер обработки команд из очереди.
func (p *Bot) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case cmd := <-p.chCommand:
			cmd.fn(ctx)
		}
	}
}

// HandlerStart — обработчик команды /start. Регистрирует пользователя.
func (p *Bot) HandlerStart(c tele.Context) error {
	return p.addCommand(c, &Command{fn: func(ctx context.Context) error {
		return db.AddUser(ctx, p.conn, c.Sender().ID, c.Chat().ID, c.Sender().Username)
	}})
}

// HandlerGet — обработчик команды /get <id>. Получает текст по ID файла.
func (p *Bot) HandlerGet(c tele.Context) error {

	args := c.Args()
	if len(args) < 1 {
		return c.Send("/get <id>")
	}

	id := args[0]

	return p.addCommand(c, &Command{fn: func(ctx context.Context) error {

		user := c.Sender().ID
		result, err := db.GetUserFileItem(ctx, p.conn, user, id)
		if err != nil {
			return err
		}

		c.Send(result)

		return err
	}})
}

// HandlerList — обработчик команды /list. Возвращает список сохраненных файлов.
func (p *Bot) HandlerList(c tele.Context) error {
	return p.addCommand(c, &Command{fn: func(ctx context.Context) error {
		result, err := db.GetUserFile(ctx, p.conn, c.Sender().ID)
		if err != nil {
			return err
		}

		c.Send(result)

		return err
	}})
}

// HandlerFind — обработчик команды /find <word>. Ищет файлы по ключевому слову.
func (p *Bot) HandlerFind(c tele.Context) error {
	args := c.Args()
	if len(args) < 1 {
		return c.Send("/find <word>")
	}

	word := args[0]
	return p.addCommand(c, &Command{fn: func(ctx context.Context) error {

		list, err := db.GetFileByWord(ctx, p.conn, c.Sender().ID, word)

		c.Send(list)
		return err
	}})
}

// Handlerchat — обработчик команды /chat <id> <вопрос>.
// Загружает контекст (транскрипцию) встречи из БД и отправляет вопрос к LLM.
func (p *Bot) Handlerchat(c tele.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return c.Send("/chat <id> <вопрос>")
	}

	id := args[0]
	question := strings.Join(args[1:], " ")

	return p.addCommand(c, &Command{fn: func(ctx context.Context) error {

		user := c.Sender().ID
		transcriptionID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return c.Send("неверный ID: " + id)
		}

		t, err := db.GetTranscriptionByID(ctx, p.conn, user, transcriptionID)
		if err != nil {
			return err
		}

		context := t.Transcription
		if context == "" {
			context = t.Summary
		}

		x, err := p.chatProcessor.GetAnswer(ctx, question, context)
		if err != nil {
			return err
		}
		return c.Send(x)
	}})
}

// HandlerOnVoice — обработчик голосовых сообщений. Добавляет задачу на распознавание.
func (p *Bot) HandlerOnVoice(c tele.Context) error {
	return p.addCommand(c, &Command{fn: func(ctx context.Context) error {
		p.speachTaskProcessor.AddTask(&model.SpeachTaskData{
			User:   c.Sender().ID,
			ChatID: c.Chat().ID,
			Name:   "voice_" + time.Now().Format("2006-01-02_15:04:05"),
			Input:  c.Message().Voice.FileReader})
		return nil
	}})
}

// HandlerOnAudio — обработчик аудиофайлов. Добавляет задачу на распознавание.
func (p *Bot) HandlerOnAudio(c tele.Context) error {
	return p.addCommand(c, &Command{fn: func(ctx context.Context) error {
		p.speachTaskProcessor.AddTask(&model.SpeachTaskData{
			User:   c.Sender().ID,
			ChatID: c.Chat().ID,
			Name:   c.Message().Audio.FileName,
			Input:  c.Message().Audio.FileReader})
		return nil
	}})
}

// HandlerOnText — обработчик текстовых сообщений. Добавляет задачу на распознавание.
func (p *Bot) HandlerOnText(c tele.Context) error {
	return p.addCommand(c, &Command{
		fn: func(ctx context.Context) error {
			p.speachTaskProcessor.AddTask(&model.SpeachTaskData{
				User:   c.Sender().ID,
				ChatID: c.Chat().ID,
				Name:   "text_" + time.Now().Format("2006-01-02_15:04:05"),
				Input:  bytes.NewReader(unsafe.Slice(unsafe.StringData(c.Message().Text), len(c.Message().Text)))})
			return nil
		}})
}
