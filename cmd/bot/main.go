package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"tg_bot_subscription_reminder/internal/config"
	"tg_bot_subscription_reminder/internal/database"
	"tg_bot_subscription_reminder/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := database.NewDB(
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer db.Close()
	log.Println(" connecting to PostgreSQL")

	userRepo := repository.NewUserRepository(db)

	bot, err := tgbotapi.NewBotAPI(cfg.TGToken)
	if err != nil {
		log.Fatalf("bot API error: %v", err)
	}
	log.Printf("authorized as @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("bot started <3")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			ctx := context.Background()
			if err := userRepo.SaveUser(ctx, update.Message.From); err != nil {
				log.Printf("error saving user: %v", err)
			}
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "save db.")
					bot.Send(msg)
				case "ping":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "db connect")
					bot.Send(msg)
				default:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "not found")
					bot.Send(msg)
				}
			}

		case <-sigChan:
			log.Println("\n shutting down gracefully...")
			bot.StopReceivingUpdates()
			return
		}
	}
}