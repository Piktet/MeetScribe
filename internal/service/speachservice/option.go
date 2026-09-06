// Package speachservice — опции для конфигурации SpeechService.
package speachservice

import (
	"time"

	"github.com/Piktet/MeetScribe/internal/repository/speach"
)

const defaultStatusTimeout = time.Second

// speachOption — параметры конфигурации для работы с SaluteSpeech.
type speachOption struct {
	connSpeach    *speach.SpeachConnection // по��ключение к SaluteSpeech API
	host          string                   // хост API
	statusTimeout time.Duration            // период опроса статуса задачи
}

// Option — функция-опция для конфигурации SpeechService.
// Применяет настройку к speachOption.
type Option func(t *speachOption)

// WithSpeach устанавливает подключение к SaluteSpeech API.
func WithSpeach(connSpeach *speach.SpeachConnection) Option {
	return func(x *speachOption) {
		x.connSpeach = connSpeach
	}
}

// WithHost устанавливает хост API.
func WithHost(host string) Option {
	return func(x *speachOption) {
		x.host = host
	}
}

// WithStatusTimeout устанавливает период опроса статуса задачи.
func WithStatusTimeout(t time.Duration) Option {
	return func(x *speachOption) {
		x.statusTimeout = t
	}
}

// newSpeachOption создает speachOption с дефолтными значениями и применяет опции.
func newSpeachOption(opts ...Option) *speachOption {
	o := &speachOption{
		statusTimeout: defaultStatusTimeout,
	}
	for _, v := range opts {
		v(o)
	}
	return o
}
