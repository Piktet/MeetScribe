// Package db — чтение данных из базы данных.
package db

import (
	"context"
	"errors"

	"github.com/Piktet/tg_bot/internal/logger"
	"github.com/Piktet/tg_bot/internal/model"

	"go.uber.org/zap"
)

const (
	// qGetUserTranscriptions — запрос списка транскрипций пользователя, отсортированных по дате (новые первые).
	qGetUserTranscriptions = `select id, name, status, created_at from transcriptions where user_id = $1 order by created_at desc`

	// qGetTranscriptionByID — запрос конкретной транскрипции по внутреннему ID (включая transcription и summary).
	qGetTranscriptionByID = `select id, user_id, chat_id, name, file_path, output_file_id, task_id, transcription, summary, status, created_at, updated_at from transcriptions where id = $1`

	// qSearchTranscriptions — полнотекстовый поиск по транскрипции и краткой выжимке пользователя.
	qSearchTranscriptions = `select id, name, status, created_at from transcriptions where user_id = $1 and (to_tsvector('russian', transcription) @@ plainto_tsquery('russian', $2) or to_tsvector('russian', summary) @@ plainto_tsquery('russian', $2)) order by created_at desc`
)

// GetUserTranscriptions возвращает список транскрипций пользователя, отсортированных по дате (новые первые).
// Возвращает только метаданные (id, name, status, created_at) — без полного текста транскрипции.
func GetUserTranscriptions(ctx context.Context, conn model.Connection, userID int64) ([]*model.Transcription, error) {
	rows, err := conn.Query(ctx, qGetUserTranscriptions, userID)
	if err != nil {
		logger.Log().Error("error GetUserTranscriptions - Query", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	transcriptions := make([]*model.Transcription, 0)

	for rows.Next() {
		var t model.Transcription
		if err := rows.Scan(&t.ID, &t.Name, &t.Status, &t.CreatedAt); err != nil {
			logger.Log().Error("error GetUserTranscriptions - Scan", zap.Error(err))
			return nil, err
		}
		t.UserID = userID
		transcriptions = append(transcriptions, &t)
	}
	if rows.Err() != nil {
		logger.Log().Error("error GetUserTranscriptions - Rows error", zap.Error(rows.Err()))
		return nil, rows.Err()
	}

	return transcriptions, nil
}

// GetTranscriptionByID возвращает полную транскрипцию по её внутреннему ID.
// Проверяет, что запись принадлежит указанному пользователю (user isolation).
func GetTranscriptionByID(ctx context.Context, conn model.Connection, userID, transcriptionID int64) (*model.Transcription, error) {
	rows, err := conn.Query(ctx, qGetTranscriptionByID, transcriptionID)
	if err != nil {
		logger.Log().Error("error GetTranscriptionByID - Query", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var t model.Transcription
	if !rows.Next() {
		return nil, errors.New("transcription not found")
	}
	if err := rows.Scan(
		&t.ID, &t.UserID, &t.ChatID, &t.Name, &t.FilePath, &t.OutputFileID,
		&t.TaskID, &t.Transcription, &t.Summary, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		logger.Log().Error("error GetTranscriptionByID - Scan", zap.Error(err))
		return nil, err
	}

	// Проверка изоляции данных: транскрипция должна принадлежать запрашивающему пользователю.
	if t.UserID != userID {
		return nil, errors.New("access denied: transcription does not belong to this user")
	}

	if rows.Err() != nil {
		logger.Log().Error("error GetTranscriptionByID - Rows error", zap.Error(rows.Err()))
		return nil, rows.Err()
	}

	return &t, nil
}

// SearchTranscriptions выполняет полнотекстовый поиск по транскрипциям и кратким выжимкам пользователя.
// Возвращает только метаданные найденных транскрипций (id, name, status, created_at).
func SearchTranscriptions(ctx context.Context, conn model.Connection, userID int64, query string) ([]*model.Transcription, error) {
	rows, err := conn.Query(ctx, qSearchTranscriptions, userID, query)
	if err != nil {
		logger.Log().Error("error SearchTranscriptions - Query", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	transcriptions := make([]*model.Transcription, 0)

	for rows.Next() {
		var t model.Transcription
		if err := rows.Scan(&t.ID, &t.Name, &t.Status, &t.CreatedAt); err != nil {
			logger.Log().Error("error SearchTranscriptions - Scan", zap.Error(err))
			return nil, err
		}
		t.UserID = userID
		transcriptions = append(transcriptions, &t)
	}
	if rows.Err() != nil {
		logger.Log().Error("error SearchTranscriptions - Rows error", zap.Error(rows.Err()))
		return nil, rows.Err()
	}

	return transcriptions, nil
}
