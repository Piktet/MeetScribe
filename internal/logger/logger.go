// Package logger — логирование с использованием zap.
package logger

import (
	"go.uber.org/zap"
)

var log *zap.Logger = zap.NewNop()

// Log возвращает экземпляр глобального логера.
// Возвращает zap.NewNop() если InitLogger ещё не был вызван.
func Log() *zap.Logger {
	return log
}

// InitLogger инициализирует глобальный логер с указанным уровнем.
// Принимает текстовый уровень логирования (например, "DEBUG", "INFO", "WARN", "ERROR").
// Возвращает ошибку, если указанный уровень невалиден.
func InitLogger(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	log = zl
	return nil
}
