// Package speachservice — сервис для распознавания речи (SaluteSpeech).
package speachservice

import (
	"context"

	"github.com/Piktet/MeetScribe/internal/model"

	"golang.org/x/sync/errgroup"
)

// SpeachService — сервис для обработки задач распознавания речи.
// Управляет очередью задач и воркерами для их выполнения.
type SpeachService struct {
	*speachOption                                 // параметры конфигурации
	chTaskRequest  chan *model.SpeachTaskData     // входящая очередь задач
	chTaskResponse chan *model.SpeachTaskResponse // исходящая очередь результатов
}

// New создает новый экземпляр SpeachService с указанными опциями.
func New(opts ...Option) *SpeachService {
	return &SpeachService{
		speachOption: newSpeachOption(opts...),
	}
}

// AddTask добавляет задачу на распознавание в очередь.
func (p *SpeachService) AddTask(x *model.SpeachTaskData) {
	p.chTaskRequest <- x
}

// GetTaskResponse возвращает канал для получения результатов задач.
func (p *SpeachService) GetTaskResponse() chan *model.SpeachTaskResponse {
	return p.chTaskResponse
}

// Start запускает воркеры обработки задач.
// cnt — количество воркеров, size — размер очередей.
func (p *SpeachService) Start(ctx context.Context, cnt, size int) error {
	p.chTaskRequest = make(chan *model.SpeachTaskData, size)
	p.chTaskResponse = make(chan *model.SpeachTaskResponse, size)
	defer close(p.chTaskRequest)
	defer close(p.chTaskResponse)
	var wg errgroup.Group
	for range cnt {
		wg.Go(func() error {
			return p.Worker(ctx)
		})
	}
	return wg.Wait()
}

// Worker — воркер обработки задач распознавания речи.
// Читает задачи из входящей очереди, обрабатывает и отправляет результаты в исходящую очередь.
func (p *SpeachService) Worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case taskData := <-p.chTaskRequest:
			task := NewTask(p.speachOption)
			if x, err := task.Process(ctx, taskData); err == nil {
				p.chTaskResponse <- x
			}
		}
	}

}
