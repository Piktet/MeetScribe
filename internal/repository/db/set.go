// Package db — запись данных в базу данных.
package db

import (
	"context"

	"github.com/Piktet/tg_bot/internal/logger"
	"github.com/Piktet/tg_bot/internal/model"

	"go.uber.org/zap"
)

const (
	// qAddUser — вставка нового пользователя (игнорирует дубли).
	qAddUser = "insert into users(id, chat_id, name, created_at) values ($1, $2, $3, current_timestamp) ON CONFLICT DO NOTHING"

	// qSaveTranscription — вставка новой транскрипции.
	qSaveTranscription = `insert into transcriptions(user_id, chat_id, name, file_path, output_file_id, task_id, transcription, summary, status)
	                      values($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// qUpdateTranscriptionStatus — обновление статуса и/или полей транскрипции.
	qUpdateTranscriptionStatus = `update transcriptions
	                               set transcription = coalesce($2, transcription),
	                                   summary       = coalesce($3, summary),
	                                   status        = coalesce($4, status),
	                                   updated_at    = current_timestamp
	                             where id = $1`
)

// AddUser регистрирует пользователя в базе данных — запоминает его идентификатор.
func AddUser(ctx context.Context, conn model.Connection, id, chatId int64, name string) error {
	if err := conn.Execute(ctx, qAddUser, id, chatId, name); err != nil {
		logger.Log().Error("error AddUser - Query", zap.Error(err))
		return err
	}
	return nil
}

// SaveTranscription сохраняет новую транскрипцию в базу данных.
func SaveTranscription(ctx context.Context, conn model.Connection, t *model.Transcription) error {
	return conn.Execute(ctx, qSaveTranscription,
		t.UserID, t.ChatID, t.Name, t.FilePath, t.OutputFileID,
		t.TaskID, t.Transcription, t.Summary, t.Status,
	)
}

// UpdateTranscriptionStatus обновляет статус, транскрипцию или краткую выжимку существующей записи.
func UpdateTranscriptionStatus(ctx context.Context, conn model.Connection, id int64, transcription, summary, status string) error {
	return conn.Execute(ctx, qUpdateTranscriptionStatus, id, transcription, summary, status)
}
