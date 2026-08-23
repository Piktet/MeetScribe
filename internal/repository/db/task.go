// Package db — операции для сервисного слоя бота.
package db

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Piktet/tg_bot/internal/logger"
	"github.com/Piktet/tg_bot/internal/model"

	"go.uber.org/zap"
)

// AddTask сохраняет результат задачи распознавания речи в базу данных.
// task — результат распознавания, short — краткая выжимка от LLM.
func AddTask(ctx context.Context, conn model.Connection, task *model.SpeachTaskResponse, short string) error {
	t := &model.Transcription{
		UserID:        task.User,
		ChatID:        task.ChatID,
		Name:          task.Name,
		FilePath:      task.InFileID,
		OutputFileID:  task.OutFileID,
		TaskID:        task.TaskID,
		Transcription: string(task.Output),
		Summary:       short,
		Status:        model.StatusDone,
	}
	if err := SaveTranscription(ctx, conn, t); err != nil {
		logger.Log().Error("error AddTask - SaveTranscription", zap.Error(err))
		return err
	}
	return nil
}

// GetUserFile возвращает список транскрипций пользователя в виде текста.
func GetUserFile(ctx context.Context, conn model.Connection, userID int64) (string, error) {
	list, err := GetUserTranscriptions(ctx, conn, userID)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "список пуст", nil
	}
	result := ""
	for _, t := range list {
		result += fmt.Sprintf("ID: %s | %s | %s | %s\n", t.ID, t.Name, t.Status, t.CreatedAt.Format("2006-01-02 15:04"))
	}
	return result, nil
}

// GetUserFileItem возвращает текст транскрипции по её ID.
func GetUserFileItem(ctx context.Context, conn model.Connection, userID int64, id string) (string, error) {
	transcriptionID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid id: %s", id)
	}
	t, err := GetTranscriptionByID(ctx, conn, userID, transcriptionID)
	if err != nil {
		return "", err
	}
	result := fmt.Sprintf("ID: %s\nНазвание: %s\nСтатус: %s\nДата: %s\n\nТранскрипция:\n%s\n\nВыжимка:\n%s",
		t.ID, t.Name, t.Status, t.CreatedAt.Format("2006-01-02 15:04"), t.Transcription, t.Summary)
	return result, nil
}

// GetFileByWord выполняет полнотекстовый поиск по транскрипциям пользователя.
func GetFileByWord(ctx context.Context, conn model.Connection, userID int64, word string) (string, error) {
	list, err := SearchTranscriptions(ctx, conn, userID, word)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "ничего не найдено", nil
	}
	result := ""
	for _, t := range list {
		result += fmt.Sprintf("ID: %s | %s | %s | %s\n", t.ID, t.Name, t.Status, t.CreatedAt.Format("2006-01-02 15:04"))
	}
	return result, nil
}
