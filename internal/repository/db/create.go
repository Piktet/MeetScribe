// Package db — создание таблиц в базе данных.
package db

import (
	"context"

	"github.com/Piktet/tg_bot/internal/model"
)

// createTablesSQL — DDL-скрипт для создания всех необходимых таблиц.
const createTablesSQL = `
-- Таблица пользователей.
create table if not exists users (
    id bigint primary key,
    chat_id bigint,
    name text,
    created_at timestamp default now()
);

-- Таблица транскрипций (встречи/тесты).
create table if not exists transcriptions (
    id            bigserial primary key,
    user_id       bigint         not null,
    chat_id       bigint         not null,
    name          text           not null,
    file_path     text,
    output_file_id text,
    task_id       uuid,
    transcription text,
    summary       text,
    status        text           not null default 'pending',
    created_at    timestamp      default now(),
    updated_at    timestamp      default now()
);

-- Индекс для быстрого поиска по user_id и статусу.
create index if not exists idx_transcriptions_user_id on transcriptions(user_id);

-- Индекс полнотекстового поиска по транскрипции.
create index if not exists idx_transcriptions_transcription on transcriptions using gin(to_tsvector('russian', transcription));

-- Индекс полнотекстового поиска по краткой выжимке.
create index if not exists idx_transcriptions_summary on transcriptions using gin(to_tsvector('russian', summary));
`

// Create создает таблицы users и transcriptions в базе данных, если они ещё не существуют.
func Create(ctx context.Context, conn model.Connection) error {
	return conn.Execute(ctx, createTablesSQL)
}
