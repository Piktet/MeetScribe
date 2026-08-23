// Package main — точка входа приложения.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Piktet/tg_bot/internal/config"
	"github.com/Piktet/tg_bot/internal/logger"
	"github.com/Piktet/tg_bot/internal/repository/chat"
	"github.com/Piktet/tg_bot/internal/repository/db"
	"github.com/Piktet/tg_bot/internal/repository/speach"
	"github.com/Piktet/tg_bot/internal/service/bot"
	"github.com/Piktet/tg_bot/internal/service/chatservice"
	"github.com/Piktet/tg_bot/internal/service/speachservice"
	"golang.org/x/sync/errgroup"
)

var (
	// buildVersion — версия сборки приложения.
	buildVersion = "N/A"
	// buildDate — дата сборки приложения.
	buildDate = "N/A"
	// buildCommit — хеш коммита сборки.
	buildCommit = "N/A"
)

const speachStreamCount = 1
const queueSpeachSize = 100

// main — точка входа приложения.
// Инициализирует конфигурацию, логгер, подключения к БД и внешним API,
// запускает сервисы бота и распознавания речи.
func main() {
	if strings.TrimSpace(buildVersion) == "" {
		buildVersion = "N/A"
	}
	if strings.TrimSpace(buildDate) == "" {
		buildDate = "N/A"
	}
	if strings.TrimSpace(buildCommit) == "" {
		buildCommit = "N/A"
	}
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	ctx, fnCancel := context.WithCancelCause(context.Background())
	defer fnCancel(nil)
	if err := create(ctx); err != nil {
		log.Fatalf("exist with error: %v", err)
	}
}

// create — инициализирует все компоненты приложения и запускает сервисы.
// Создает подключения к БД, Speech API и Chat API,
// запускает воркеры для обработки задач распознавания речи и бота.
func create(ctx context.Context) error {

	config := config.New()
	if err := logger.InitLogger(config.GetLogLevel()); err != nil {
		panic(err)
	}

	dbConn, err := db.New(config.GetConnectionString())
	if err != nil {
		return err
	}
	if err := db.Create(ctx, dbConn); err != nil {
		return err
	}

	speachConn := speach.New(config.GetSpeachAuthHost(), config.GetSpeachRQUID(), config.GetSpeachAuthKey())
	if err := speachConn.Connect(ctx); err != nil {
		return err
	}
	speachService := speachservice.New(
		speachservice.WithSpeach(speachConn),
		speachservice.WithHost(config.GetSpeachRequestHost()),
	)

	chatConn := chat.New(config.GetChatAuthHost(), config.GetChatRQUID(), config.GetChatAuthKey())
	if err := chatConn.Connect(ctx); err != nil {
		return err
	}
	chatService := chatservice.New(config.GetChatRequestHost(), chatConn)

	botService := bot.New(config.GetBotToken(), dbConn, speachService, chatService)

	wg, sendCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		return speachService.Start(sendCtx, speachStreamCount, queueSpeachSize)
	})
	wg.Go(func() error {
		return botService.Start(sendCtx, speachStreamCount, queueSpeachSize)
	})

	return wg.Wait()
}
