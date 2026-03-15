CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT PRIMARY KEY,             
    username VARCHAR(255),                  
    first_name VARCHAR(255),                
    last_name VARCHAR(255),                 
    timezone VARCHAR(50) DEFAULT 'UTC',     
    language_code VARCHAR(10) DEFAULT 'ru', 
    is_blocked BOOLEAN DEFAULT FALSE,       
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_timezone ON users(timezone);

CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_blocked) WHERE is_blocked = FALSE;

CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,                  
    user_id BIGINT NOT NULL,                
    name VARCHAR(255) NOT NULL,             
    amount NUMERIC(10, 2) NOT NULL,         
    currency VARCHAR(10) DEFAULT 'RUB',     
    next_payment_date TIMESTAMP WITH TIME ZONE NOT NULL,
    period_days INTEGER DEFAULT 30,         
    is_active BOOLEAN DEFAULT TRUE,         
    notification_enabled BOOLEAN DEFAULT TRUE,
    last_notified_at TIMESTAMP WITH TIME ZONE, 
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_subscriptions_user 
        FOREIGN KEY (user_id) 
        REFERENCES users(user_id) 
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_next_payment 
    ON subscriptions(next_payment_date) 
    WHERE is_active = TRUE AND notification_enabled = TRUE;


CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id 
    ON subscriptions(user_id);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_active 
    ON subscriptions(user_id, is_active) 
    WHERE is_active = TRUE;


CREATE TABLE IF NOT EXISTS notification_log (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    subscription_id INTEGER,
    notification_type VARCHAR(50),          
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    success BOOLEAN DEFAULT TRUE,           
    error_message TEXT,                     
    
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