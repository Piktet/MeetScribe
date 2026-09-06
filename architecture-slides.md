# MeetScribe — Архитектура проекта

---

## 1. Что такое MeetScribe?

**Telegram-бот для распознавания речи и анализа аудиоконтента**

- Принимает голосовые сообщения, аудиофайлы и тексты
- Распознаёт речь через **SaluteSpeech API** (Сбер)
- Генерирует краткие выжимки через **GigaChat API** (Сбер)
- Позволяет искать и задавать вопросы по сохранённым транскрипциям

---

## 2. Технологический стек

| Компонент | Технология |
|-----------|-----------|
| Язык | Go 1.26 |
| Telegram API | `gopkg.in/telebot.v3` |
| База данных | PostgreSQL + `jackc/pgx/v5` |
| Распознавание речи | SaluteSpeech API (Sber) |
| LLM | GigaChat API (Sber) |
| Логирование | `log/slog` |
| Конкурентность | `golang.org/x/sync/errgroup` |

---

## 3. Структура проекта

```
cmd/
  └── main.go              ← Точка входа, инициализация

internal/
  ├── config/              ← Конфигурация (JSON + env + flags)
  ├── logger/              ← Логирование на slog
  ├── model/               ← Доменные модели и интерфейсы
  ├── repository/
  │   ├── db/              ← PostgreSQL (pgx)
  │   ├── speach/          ← SaluteSpeech API
  │   └── chat/            ← GigaChat API
  └── service/
      ├── bot/             ← Telegram-бот
      ├── speachservice/   ← Распознавание речи
      └── chatservice/     ← Запросы к LLM
```

---

## 4. Слои архитектуры

```
┌─────────────────────────────────────────┐
│           cmd/main.go                   │  ← Оркестрация
├─────────────────────────────────────────┤
│        service/ (бизнес-логика)         │  ← Сервисы
├─────────────────────────────────────────┤
│      repository/ (внешние системы)      │  ← Репозитории
├─────────────────────────────────────────┤
│          model/ (модели + интерфейсы)   │  ← Домен
└─────────────────────────────────────────┘
```

**Принцип:** верхние слои зависят от нижних через интерфейсы.

---

## 5. Модель данных

### Таблица `users`
```
id (bigint)      — Telegram user ID
chat_id (bigint) — Telegram chat ID
name (text)      — имя пользователя
created_at       — дата регистрации
```

### Таблица `transcriptions`
```
id               — первичный ключ
user_id          — владелец
name             — название
file_path        — ID файла в SaluteSpeech
task_id          — ID задачи в SaluteSpeech
transcription    — полный текст
summary          — выжимка от LLM
status           — PENDING / UPLOADING / PROCESSING / DONE / ERROR
created_at       — дата создания
updated_at       — дата обновления
```

---

## 6. Индексы базы данных

```sql
-- Быстрый поиск по пользователю
idx_transcriptions_user_id ON transcriptions(user_id)

-- Полнотекстовый поиск по транскрипции (русский)
idx_transcriptions_transcription USING gin(
  to_tsvector('russian', transcription)
)

-- Полнотекстовый поиск по выжимке (русский)
idx_transcriptions_summary USING gin(
  to_tsvector('russian', summary)
)
```

---

## 7. Команды бота

| Команда | Описание |
|---------|----------|
| `/start` | Регистрация пользователя |
| `/list` | Список всех транскрипций |
| `/get <id>` | Полный текст транскрипции |
| `/find <word>` | Поиск по ключевому слову |
| `/chat <id> <вопрос>` | Вопрос по конкретной транскрипции |
| 🎤 Голосовое сообщение | Автоматическое распознавание |
| 🎵 Аудиофайл | Автоматическое распознавание |

---

## 8. Архитектура обработки

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Telegram Bot │────▶│  Command     │────▶│  Speech      │
│  (LongPoller)│     │  Queue       │     │  Service     │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                  │
                                          ┌───────▼───────┐
                                          │   Worker      │
                                          │  (1 поток)    │
                                          └───────┬───────┘
                                                  │
                                    ┌─────────────┼─────────────┐
                                    │             │             │
                              ┌─────▼─────┐ ┌────▼────┐ ┌─────▼─────┐
                              │  Upload   │ │ Create  │ │ Download  │
                              │  Audio    │ │  Task   │ │  Result   │
                              └─────┬─────┘ └────┬────┘ └─────┬─────┘
                                    │            │             │
                              ┌─────▼────────────▼─────────────▼─────┐
                              │        SaluteSpeech API               │
                              └──────────────────────────────────────┘
```

---

## 9. Поток данных

```
Пользователь отправляет голосовое
         │
         ▼
  HandlerOnVoice
         │
         ▼
  chCommand → bot.worker()
         │
         ▼
  chTaskRequest → speach.Worker()
         │
         ▼
  1. Upload(audio)       → fileID
  2. CreateTask(fileID)  → taskID
  3. GetStatus(taskID)   → цикл опроса
  4. Download(fileID)    → транскрипция
         │
         ▼
  chTaskResponse
         │
         ▼
  getTaskResultProcess()
         │
         ├── GetShort(transcription) → summary (GigaChat)
         └── db.AddTask() → INSERT в PostgreSQL
```

---

## 10. Сервисы и их взаимодействие

```
┌──────────────────────────────────────────────────────┐
│                    main.go                           │
│                                                      │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐    │
│  │   config   │  │   logger   │  │    db      │    │
│  └─────┬──────┘  └────────────┘  └─────┬──────┘    │
│        │                                │           │
│  ┌─────▼────────────────────────────────▼──────┐    │
│  │          Bot Service                        │    │
│  │  ┌────────────┐  ┌────────────────────┐    │    │
│  │  │ Commands   │  │ getTaskResult      │    │    │
│  │  │ Worker     │  │ Process            │    │    │
│  │  └────────────┘  └────────┬───────────┘    │    │
│  └────────────────────────────┼────────────────┘    │
│                               │                      │
│              ┌────────────────┼────────────┐        │
│              │                │             │        │
│      ┌───────▼──────┐ ┌──────▼──────┐ ┌────▼──────┐ │
│      │ Speech       │ │  Chat       │ │   DB      │ │
│      │ Service      │ │  Service    │ │ Repository│ │
│      └───────┬──────┘ └──────┬──────┘ └───────────┘ │
│              │               │                       │
│      ┌───────▼──────┐ ┌─────▼────────┐              │
│      │ SaluteSpeech │ │  GigaChat    │              │
│      │   API        │ │    API       │              │
│      └──────────────┘ └──────────────┘              │
└──────────────────────────────────────────────────────┘
```

---

## 11. Конкурентность

```
errgroup.WithContext(ctx)
  │
  ├── speachService.Start(ctx, 1, 100)
  │     ├── chTaskRequest  (chan, буфер 100)
  │     ├── chTaskResponse (chan, буфер 100)
  │     └── 1 Worker goroutine
  │
  └── botService.Start(ctx, 1, 100)
        ├── chCommand (chan, буфер 100)
        ├── 1 Command Worker goroutine
        ├── Telegram LongPoller (10s timeout)
        └── getTaskResultProcess goroutine
```

**При отмене контекста** — все goroutine завершаются через `ctx.Done()`.

---

## 12. Управление токенами

```
SpeachConnection / ChatConnection
  │
  ├── Connect() → POST /api/v2/oauth
  │     ├── Получает access_token + expires_at
  │     └── time.AfterFunc(до истечения) → Connect()
  │
  └── GetToken() → возвращает текущий token
```

**Автоматическое обновление** — токен обновляется до истечения срока действия.

---

## 13. Конфигурация

```
Приоритет источников (от высшего к низшему):

  1. Флаги командной строки
     -a, -b, -l, -config, -db, ...

  2. Переменные окружения
     SPEECH_AUTH_ADDRESS, BOT_TOKEN, ...

  3. JSON-файл
     config.json

  4. Значения по умолчанию
     ngw.devices.sberbank.ru:9443, ...
```

---

## 14. Ключевые особенности

- **Чистая архитектура** — разделение на repository/service/model
- **Интерфейсы** — `SpeechClient`, `LLMClient`, `Connection` для тестируемости
- **Полнотекстовый поиск** — PostgreSQL full-text search с русским словарём
- **Изоляция данных** — проверка `UserID` при доступе к транскрипциям
- **Асинхронная обработка** — очередь задач с воркерами
- **Graceful shutdown** — отмена контекста завершает все goroutine

---

## 15. Зоны роста

| Приоритет | Проблема |
|-----------|----------|
| 🔴 Критический | Секреты в `config.json` в репозитории |
| 🔴 Критический | Нет проверки `nil` после `tele.NewBot()` |
| 🟡 Средний | `HandlerOnText` перехватывает команды |
| 🟡 Средний | Нет mock-тестов для service/repo слоёв |
| 🟡 Средний | Нет graceful shutdown по SIGINT |
| 🟢 Низкий | Опечатка `speach` → `speech` |
| 🟢 Низкий | Минимальный README |

---

## 16. Итоги

```
MeetScribe = Telegram + SaluteSpeech + GigaChat + PostgreSQL
```

**Что умеет:**
- Распознавание речи из голосовых и аудио
- Генерация выжимек через LLM
- Поиск и вопросы по транскрипциям

**Архитектура:**
- Чистое разделение на слои
- Интерфейсы для тестируемости
- Асинхронная обработка с очередями
- Автоматическое управление токенами

**Что улучшить:**
- Безопасность (секреты)
- Тесты (покрытие)
- Обработка ошибок
- Graceful shutdown
