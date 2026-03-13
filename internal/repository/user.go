package repository

import (
	"context"
	"log"
	"tg_bot_subscription_reminder/internal/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) SaveUser(ctx context.Context, user *tgbotapi.User) error {
	query := `
		INSERT INTO users (user_id, username, first_name, last_name, language_code)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = NOW()
	`
	
	_, err := r.db.Pool.Exec(ctx, query,
		user.ID,
		user.UserName,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
	)
	
	if err != nil {
		return err
	}
	
	log.Printf("user saved: ID=%d, Username=%s", user.ID, user.UserName)
	return nil
}