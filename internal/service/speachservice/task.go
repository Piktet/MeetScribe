// Package speachservice — обработка одной задачи распознавания речи.
package speachservice

import (
	"context"
	"errors"
	"time"

	"github.com/Piktet/tg_bot/internal/model"
	"github.com/Piktet/tg_bot/internal/repository/speach"
)

// SpeachTask — задача распознавания речи.
// Выполняет полный цикл: загрузка аудио, создание задачи, мониторинг статуса, скачивание результата.
type SpeachTask struct {
	*speachOption        // параметры конфигурации
	token         string // токен авторизации
	inFileID      string // ID загруженного входного файла
	outFileID     string // ID файла с результатом
	taskID        string // ID задачи в SaluteSpeech
}

// NewTask создает новую задачу распознавания речи.
func NewTask(opt *speachOption) *SpeachTask {
	return &SpeachTask{
		speachOption: opt,
	}
}

// Process выполняет полный цикл обработки задачи распознавания речи.
// Загружает аудио, создает задачу, ожидает завершения, скачивает результат.
func (p *SpeachTask) Process(ctx context.Context, task *model.SpeachTaskData) (*model.SpeachTaskResponse, error) {
	token, err := p.connSpeach.GetToken()
	if err != nil {
		return nil, err
	}

	p.token = token

	inFileID, err := speach.Upload(ctx, p.host, p.token, task.Input)
	if err != nil {
		return nil, err
	}

	p.inFileID = inFileID

	taskID, outFileID, status, err := speach.CreateTask(ctx, p.host, p.token, inFileID)
	if err != nil {
		return nil, err
	}
	p.taskID = taskID
	p.outFileID = outFileID

	next, err := checkSpeachStatus(status)

	if err != nil {
		return nil, err
	}

	if next {
		if err := p.checkSpeachStatus(ctx); err != nil {
			return nil, err
		}
	}

	x, err := speach.Download(ctx, p.host, p.token, p.taskID)
	if err != nil {
		return nil, err
	}

	return &model.SpeachTaskResponse{
		SpeachTaskData: task,
		InFileID:       inFileID,
		OutFileID:      outFileID,
		TaskID:         taskID,
		Output:         x,
	}, nil
}

// checkSpeachStatus — мониторинг статуса задачи с периодической проверкой.
func (p *SpeachTask) checkSpeachStatus(ctx context.Context) error {
	t := time.NewTicker(p.statusTimeout)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			fileID, status, isRetry, err := speach.GetStatus(ctx, p.host, p.token, p.taskID)
			if err != nil && !isRetry {
				return err
			}

			next, err := checkSpeachStatus(status)

			if err != nil {
				return err
			}
			if !next {
				p.outFileID = fileID
				return nil
			}
		}
	}
}

// checkSpeachStatus — проверка статуса задачи распознавания.
// Возвращает (needRetry, error): needRetry = true означает, что нужно продолжать опрос.
func checkSpeachStatus(x model.ResultStatusType) (bool, error) {
	switch x {
	case model.SpeachResultStatusCanceled:
		return false, errors.New("status canceled")
	case model.SpeachResultStatusDone:
		return false, nil
	case model.SpeachResultStatusError:
		return false, errors.New("status error")
	case model.SpeachResultStatusRunning, model.SpeachResultStatusNew:
		return true, nil
	case model.SpeachResultStatusEmpty:
		return false, errors.New("status empty")
	default:
		return false, errors.New("status unknown")
	}
}
