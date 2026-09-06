# MeetScribe — Архитектура проекта

---

## Слайд 1. Обзор и структура

### MeetScribe — Telegram-бот для распознавания речи

**Стек:** Go 1.26 · PostgreSQL · SaluteSpeech API · GigaChat API · telebot.v3

**Что делает:** принимает голосовые/аудио → распознаёт речь → генерирует выжимку через LLM → позволяет искать и задавать вопросы

```
cmd/main.go  — точка входа, инициализация всех компонентов
    │
    ├── internal/config/     — конфигурация (flags > env > JSON > defaults)
    ├── internal/logger/     — логирование (slog)
    ├── internal/model/      — доменные модели + интерфейсы
    │
    ├── internal/repository/ — работа с внешними системами
    │   ├── db/              — PostgreSQL (pgx)
    │   ├── speach/          — SaluteSpeech API (загрузка, создание задачи, опрос, скачивание)
    │   └── chat/            — GigaChat API (авторизация, запросы к LLM)
    │
    └── internal/service/    — бизнес-логика
        ├── bot/             — Telegram-бот, команды, воркеры
        ├── speachservice/   — очередь задач распознавания + воркеры
        └── chatservice/     — обёртка над GigaChat API
```

**Принцип:** верхние слои зависят от нижних через интерфейсы (`SpeechClient`, `LLMClient`, `Connection`).

---

## Слайд 2. Поток данных

```
Пользователь отправляет голосовое сообщение
                    │
                    ▼
        ┌───────────────────────┐
        │  Telegram Bot         │  HandlerOnVoice
        │  LongPoller (10s)     │
        └───────────┬───────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │  chCommand (buf 100)  │  → bot.worker()
        └───────────┬───────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │  chTaskRequest (buf)  │  → speach.Worker()
        └───────────┬───────────┘
                    │
                    ▼
         ┌────────────────────────┐
         │   SpeachTask.Process() │
         │                        │
         │  1. Upload(audio)      │──→ fileID
         │  2. CreateTask(fileID) │──→ taskID
         │  3. GetStatus(taskID)  │──→ цикл опроса каждую секунду
         │  4. Download(fileID)   │──→ транскрипция
         └────────┬───────────────┘
                  │
                  ▼
        ┌───────────────────────┐
        │  chTaskResponse       │  → getTaskResultProcess()
        └───────────┬───────────┘
                    │
          ┌─────────┴─────────┐
          │                   │
          ▼                   ▼
   GigaChat API          PostgreSQL
   GetShort()            INSERT INTO
   → summary             transcriptions
```

**Итого:** 1 воркер распознавания + LongPoller Telegram + 1 воркер команд + обработка результатов.
