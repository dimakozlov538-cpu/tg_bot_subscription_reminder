-- ============================================
-- Схема базы данных для Telegram Bot: Subscription Reminder
-- DB: PostgreSQL 15+
-- ============================================

-- 1. Таблица пользователей (Users)
-- Хранит информацию о пользователях бота и их настройки
CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT PRIMARY KEY,             -- Telegram User ID (уникальный)
    username VARCHAR(255),                  -- Юзернейм в Telegram (может меняться)
    first_name VARCHAR(255),                -- Имя
    last_name VARCHAR(255),                 -- Фамилия
    timezone VARCHAR(50) DEFAULT 'UTC',     -- Часовой пояс (напр. 'Europe/Moscow')
    language_code VARCHAR(10) DEFAULT 'ru', -- Предпочитаемый язык
    is_blocked BOOLEAN DEFAULT FALSE,       -- Если пользователь заблокировал бота
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индекс для быстрого поиска по timezone (для выборки пользователей в нужное время)
CREATE INDEX IF NOT EXISTS idx_users_timezone ON users(timezone);
-- Индекс для выборки активных пользователей
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_blocked) WHERE is_blocked = FALSE;


-- 2. Таблица подписок (Subscriptions)
-- Хранит информацию о подписках каждого пользователя
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,                  -- Внутренний ID записи
    user_id BIGINT NOT NULL,                -- Ссылка на пользователя
    name VARCHAR(255) NOT NULL,             -- Название (Netflix, YouTube и т.д.)
    amount NUMERIC(10, 2) NOT NULL,         -- Стоимость (например, 299.90)
    currency VARCHAR(10) DEFAULT 'RUB',     -- Валюта (RUB, USD, EUR)
    next_payment_date TIMESTAMP WITH TIME ZONE NOT NULL, -- Дата следующего списания
    period_days INTEGER DEFAULT 30,         -- Периодичность в днях (30, 90, 365)
    is_active BOOLEAN DEFAULT TRUE,         -- Активна ли подписка
    notification_enabled BOOLEAN DEFAULT TRUE, -- Включены ли уведомления для этой подписки
    last_notified_at TIMESTAMP WITH TIME ZONE, -- Когда было последнее напоминание
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Внешний ключ с каскадным удалением (если удалим юзера, удалятся и подписки)
    CONSTRAINT fk_subscriptions_user 
        FOREIGN KEY (user_id) 
        REFERENCES users(user_id) 
        ON DELETE CASCADE
);

-- Индексы для ускорения работы планировщика (самая важная часть!)
-- 1. Быстрый поиск подписок, у которых скоро платеж
CREATE INDEX IF NOT EXISTS idx_subscriptions_next_payment 
    ON subscriptions(next_payment_date) 
    WHERE is_active = TRUE AND notification_enabled = TRUE;

-- 2. Быстрый поиск всех подписок конкретного пользователя
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id 
    ON subscriptions(user_id);

-- 3. Поиск активных подписок пользователя
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_active 
    ON subscriptions(user_id, is_active) 
    WHERE is_active = TRUE;


-- 3. Таблица истории уведомлений (Notification Log)
-- Полезно для отладки и статистики (сколько уведомлений отправлено)
CREATE TABLE IF NOT EXISTS notification_log (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    subscription_id INTEGER,
    notification_type VARCHAR(50),          -- 'reminder_3_days', 'reminder_1_day', 'payment_day'
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    success BOOLEAN DEFAULT TRUE,           -- Успешно ли отправлено
    error_message TEXT,                     -- Текст ошибки, если не удалось
    
    CONSTRAINT fk_notification_user 
        FOREIGN KEY (user_id) 
        REFERENCES users(user_id) 
        ON DELETE CASCADE,
    CONSTRAINT fk_notification_subscription 
        FOREIGN KEY (subscription_id) 
        REFERENCES subscriptions(id) 
        ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_log_user 
    ON notification_log(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_log_sent_at 
    ON notification_log(sent_at);


-- 4. Триггер для автоматического обновления updated_at
-- Чтобы не писать это вручную в каждом UPDATE запросе в Go
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at 
    BEFORE UPDATE ON subscriptions 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();


-- ============================================
-- Примеры данных для тестирования (можно удалить)
-- ============================================

-- INSERT INTO users (user_id, username, first_name, timezone) 
-- VALUES (123456789, 'testuser', 'Test', 'Europe/Moscow');

-- INSERT INTO subscriptions (user_id, name, amount, currency, next_payment_date, period_days) 
-- VALUES 
-- (123456789, 'Netflix', 799.00, 'RUB', NOW() + INTERVAL '3 days', 30),
-- (123456789, 'YouTube Premium', 199.00, 'RUB', NOW() + INTERVAL '1 day', 30),
-- (123456789, 'Spotify', 169.00, 'RUB', NOW() + INTERVAL '10 days', 30);