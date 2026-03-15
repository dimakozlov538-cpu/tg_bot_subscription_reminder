package repository

import (
	"context"
	"fmt"
	"log"
	"tg_bot_subscription_reminder/internal/database"
	"tg_bot_subscription_reminder/internal/models"
)

type SubscriptionRepository struct {
	db *database.DB
}

func NewSubscriptionRepository(db *database.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *models.Subscription) error {
	query := `
		INSERT INTO subscriptions (user_id, name, amount, currency, next_payment_date, period_days)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.Pool.QueryRow(ctx, query,
		sub.UserID,
		sub.Name,
		sub.Amount,
		sub.Currency,
		sub.NextPaymentDate,
		sub.PeriodDays,
	).Scan(&sub.ID)

	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	log.Printf("💾 Subscription created: ID=%d, Name=%s, Amount=%.2f %s", 
		sub.ID, sub.Name, sub.Amount, sub.Currency)
	return nil
}

func (r *SubscriptionRepository) GetByUserID(ctx context.Context, userID int64) ([]models.Subscription, error) {
	query := `
		SELECT id, user_id, name, amount, currency, next_payment_date, 
		       period_days, is_active, notification_enabled, 
		       last_notified_at, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY next_payment_date ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var sub models.Subscription
		err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Name,
			&sub.Amount,
			&sub.Currency,
			&sub.NextPaymentDate,
			&sub.PeriodDays,
			&sub.IsActive,
			&sub.NotificationEnabled,
			&sub.LastNotifiedAt,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		subs = append(subs, sub)
	}

	return subs, rows.Err()
}

func (r *SubscriptionRepository) Delete(ctx context.Context, userID, subID int64) error {
	query := `
		UPDATE subscriptions 
		SET is_active = FALSE, updated_at = NOW() 
		WHERE id = $1 AND user_id = $2
	`
	result, err := r.db.Pool.Exec(ctx, query, subID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}

	log.Printf("🗑️ Subscription deleted: ID=%d, UserID=%d", subID, userID)
	return nil
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, userID, subID int64) (*models.Subscription, error) {
	query := `
		SELECT id, user_id, name, amount, currency, next_payment_date, 
		       period_days, is_active, notification_enabled, 
		       last_notified_at, created_at, updated_at
		FROM subscriptions
		WHERE id = $1 AND user_id = $2
	`

	var sub models.Subscription
	err := r.db.Pool.QueryRow(ctx, query, subID, userID).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Name,
		&sub.Amount,
		&sub.Currency,
		&sub.NextPaymentDate,
		&sub.PeriodDays,
		&sub.IsActive,
		&sub.NotificationEnabled,
		&sub.LastNotifiedAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return &sub, nil
}