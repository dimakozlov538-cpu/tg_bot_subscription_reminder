package models

import "time"

type Subscription struct {
	ID	int64	`db:"id"`
	UserID	int64   `db:"user_id"`
	Name	string	`db:"name"`
	Amount	float64	`db:"amount"`
	Currency	string	`db:"currency"`
	NextPaymentDate	time.Time	`db:"next_payment_date"`
	PeriodDays	int		`db:"period_days"`
	IsActive	bool	`db:"is_active"`
	NotificationEnabled	bool	`db:"notification_enabled"`
	LastNotifiedAt	*time.Time	`db:"last_notified_at"`
	CreatedAt	time.Time 	`db:"created_at"`
	UpdatedAt	time.Time	`db:"updated_at"`
}