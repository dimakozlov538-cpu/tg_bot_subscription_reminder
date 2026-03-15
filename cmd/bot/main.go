package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"tg_bot_subscription_reminder/internal/config"
	"tg_bot_subscription_reminder/internal/database"
	"tg_bot_subscription_reminder/internal/models"
	"tg_bot_subscription_reminder/internal/repository"
)

type UserState struct {
	Step            int
	Name            string
	Amount          float64
	Currency        string
	LastPaymentDate string
	PeriodDays      int
}

var userStates = make(map[int64]*UserState)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	db, err := database.NewDB(
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)
	if err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer db.Close()
	log.Println("✅ Connecting to PostgreSQL")

	userRepo := repository.NewUserRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)

	bot, err := tgbotapi.NewBotAPI(cfg.TGToken)
	if err != nil {
		log.Fatalf("Bot API error: %v", err)
	}
	log.Printf("✅ Authorized as @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	log.Println("🚀 Bot started <3")

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
				log.Printf("❌ Error saving user: %v", err)
			}

			userID := update.Message.From.ID
			if state, exists := userStates[userID]; exists {
				handleSubscriptionFlow(bot, update.Message, state, subRepo, ctx)
				continue
			}

			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handleStart(bot, update.Message)
				case "add":
					handleAddStart(bot, update.Message, userID)
				case "list":
					handleList(bot, update.Message, subRepo, ctx, userID)
				case "expenses":
					handleExpenses(bot, update.Message, subRepo, ctx, userID)
				case "help":
					handleHelp(bot, update.Message)
				case "delete":
					handleDelete(bot, update.Message, subRepo, ctx, userID)
				default:
					sendMessage(bot, update.Message.Chat.ID, "❓ Неизвестная команда. Используйте /help")
				}
			}

		case <-sigChan:
			log.Println("\n🛑 Shutting down gracefully...")
			bot.StopReceivingUpdates()
			return
		}
	}
}

func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	text := `👋 Привет! Я бот напоминаний о подписках.

		📋 Доступные команды:
		/add — добавить подписку
		/list — показать все подписки
		/expenses — показать статистику трат
		/help — помощь

		Напиши /add чтобы добавить первую подписку!`
	sendMessage(bot, message.Chat.ID, text)
}

func handleHelp(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	text := `📖 Помощь по боту

	📋 Доступные команды:
	/start — Запустить бота
	/add — Добавить новую подписку (пошаговый мастер)
	/list — Показать все активные подписки
	/expenses — Показать статистику трат
	/delete <ID> — Удалить подписку по ID
	/help — Показать эту справку

	💡 Пример использования:
	1. Напишите /add
	2. Следуйте инструкциям бота
	3. Используйте /list для просмотра
	4. Используйте /expenses для статистики трат
	5. Используйте /delete 1 для удаления подписки с ID 1`
	sendMessage(bot, message.Chat.ID, text)
}

func handleAddStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message, userID int64) {
	userStates[userID] = &UserState{
		Step: 1,
	}

	text := `📝 Добавление новой подписки

	Шаг 1/5: Введите название подписки (например, Netflix, YouTube Premium)`
	sendMessage(bot, message.Chat.ID, text)
}

func handleSubscriptionFlow(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *UserState, subRepo *repository.SubscriptionRepository, ctx context.Context) {
	userID := message.From.ID
	chatID := message.Chat.ID

	switch state.Step {
	case 1:
		state.Name = message.Text
		state.Step = 2

		text := fmt.Sprintf(`✅ Название: %s

	Шаг 2/5: Введите стоимость подписки (число, например, 799)`, state.Name)
		sendMessage(bot, chatID, text)

	case 2:
		amount, err := strconv.ParseFloat(message.Text, 64)
		if err != nil || amount <= 0 {
			sendMessage(bot, chatID, "❌ Неверный формат! Введите число больше 0.")
			return
		}

		state.Amount = amount
		state.Step = 3

		text := fmt.Sprintf(`✅ Стоимость: %.2f

	Шаг 3/5: Введите валюту (RUB, USD, EUR)`, state.Amount)
		sendMessage(bot, chatID, text)

	case 3:
		state.Currency = message.Text
		state.Step = 4

		text := `✅ Валюта: ` + state.Currency + `

	Шаг 4/5: Введите дату последнего платежа в формате ДД.ММ.ГГГГ
	Пример: 12.03.2026

	Это дата, когда вы фактически оплатили подписку.`
		sendMessage(bot, chatID, text)

	case 4:
		parsedDate, err := time.Parse("02.01.2006", message.Text)
		if err != nil {
			sendMessage(bot, chatID, "❌ Неверный формат даты! Используйте ДД.ММ.ГГГГ\nПример: 12.03.2026")
			return
		}

		state.LastPaymentDate = parsedDate.Format("02.01.2006")
		state.Step = 5

		text := `✅ Дата последнего платежа: ` + state.LastPaymentDate + `

	Шаг 5/5: Введите период оплаты в днях (например, 30 для ежемесячной, 365 для годовой)

	Бот автоматически рассчитает дату следующего платежа.`
		sendMessage(bot, chatID, text)

	case 5:
		periodDays, err := strconv.Atoi(message.Text)
		if err != nil || periodDays <= 0 {
			sendMessage(bot, chatID, "❌ Неверный формат! Введите число больше 0.")
			return
		}

		state.PeriodDays = periodDays

		lastPaymentDate, _ := time.Parse("02.01.2006", state.LastPaymentDate)
		nextPaymentDate := lastPaymentDate.AddDate(0, 0, state.PeriodDays)

		sub := &models.Subscription{
			UserID:          userID,
			Name:            state.Name,
			Amount:          state.Amount,
			Currency:        state.Currency,
			NextPaymentDate: nextPaymentDate,
			PeriodDays:      state.PeriodDays,
			IsActive:        true,
		}

		if err := subRepo.Create(ctx, sub); err != nil {
			sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка при создании: %v", err))
			delete(userStates, userID)
			return
		}

		delete(userStates, userID)

		text := fmt.Sprintf(`✅ Подписка успешно добавлена!

		📋 Детали:
		• Название: %s
		• Стоимость: %.2f %s
		• Последний платёж: %s
		• Следующий платёж: %s
		• Период: %d дней

		Используй /list чтобы посмотреть все подписки`,
			sub.Name, sub.Amount, sub.Currency,
			lastPaymentDate.Format("02.01.2006"),
			nextPaymentDate.Format("02.01.2006"),
			sub.PeriodDays)
		sendMessage(bot, chatID, text)
	}
}

func handleList(bot *tgbotapi.BotAPI, message *tgbotapi.Message, subRepo *repository.SubscriptionRepository, ctx context.Context, userID int64) {
	subs, err := subRepo.GetByUserID(ctx, userID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
		return
	}

	if len(subs) == 0 {
		sendMessage(bot, message.Chat.ID, "📭 У вас нет активных подписок\n\nИспользуй /add чтобы добавить первую!")
		return
	}

	var text string
	text = "📋 Ваши подписки:\n\n"

	for i, sub := range subs {
		text += fmt.Sprintf("%d. %s\n", i+1, sub.Name)
		text += fmt.Sprintf("   💰 %.2f %s\n", sub.Amount, sub.Currency)
		text += fmt.Sprintf("   📅 Следующий платёж: %s\n", sub.NextPaymentDate.Format("02.01.2006"))
		text += fmt.Sprintf("   🔁 Период: %d дней\n", sub.PeriodDays)
		text += fmt.Sprintf("   ID: %d\n\n", sub.ID)
	}

	text += "Для удаления используйте: /delete 1"

	sendMessage(bot, message.Chat.ID, text)
}

func handleExpenses(bot *tgbotapi.BotAPI, message *tgbotapi.Message, subRepo *repository.SubscriptionRepository, ctx context.Context, userID int64) {
	subs, err := subRepo.GetByUserID(ctx, userID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
		return
	}

	if len(subs) == 0 {
		sendMessage(bot, message.Chat.ID, "📭 У вас нет активных подписок\n\nИспользуй /add чтобы добавить первую!")
		return
	}

	var nextPaymentTotal float64
	var annualTotal float64
	currencies := make(map[string]bool)

	for _, sub := range subs {
		nextPaymentTotal += sub.Amount
		paymentsPerYear := float64(365) / float64(sub.PeriodDays)
		annualTotal += sub.Amount * paymentsPerYear
		currencies[sub.Currency] = true
	}

	var currencyList string
	for curr := range currencies {
		currencyList += curr + " "
	}

	text := fmt.Sprintf(`💰 Статистика трат

	📅 На следующие платежи:
	• Итого: %.2f %s

	📊 Прогноз за год:
	• Итого за год: %.2f %s

	📋 Подписок учтено: %d

	Примечание: Суммы посчитаны по всем валютам (%s).
	Для точного бюджетирования рекомендуем привести к одной валюте.`,
		nextPaymentTotal, currencyList,
		annualTotal, currencyList,
		len(subs), currencyList)

	sendMessage(bot, message.Chat.ID, text)
}

func handleDelete(bot *tgbotapi.BotAPI, message *tgbotapi.Message, subRepo *repository.SubscriptionRepository, ctx context.Context, userID int64) {
	args := message.CommandArguments()

	if args == "" {
		sendMessage(bot, message.Chat.ID, "❌ Укажите ID подписки\n\nПример: /delete 1\n\nИспользуйте /list чтобы узнать ID")
		return
	}

	subID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ ID должен быть числом\n\nПример: /delete 1")
		return
	}

	if err := subRepo.Delete(ctx, userID, subID); err != nil {
		sendMessage(bot, message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
		return
	}

	sendMessage(bot, message.Chat.ID, fmt.Sprintf("✅ Подписка #%d успешно удалена", subID))
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}