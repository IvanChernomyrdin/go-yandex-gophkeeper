CREATE TYPE secret_type AS ENUM (
    'login_password',
    'text',
    'binary',
    'bank_card',
    'otp'
);

CREATE TABLE secrets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    type            secret_type NOT NULL,
    title           TEXT NOT NULL,
    payload         BYTEA NOT NULL,
    meta            TEXT,

    version         INTEGER NOT NULL DEFAULT 1,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Быстро получить все секреты пользователя
CREATE INDEX idx_secrets_user_id ON secrets(user_id);

-- Составной индекс для поиска изменившихся записей по пользователю
CREATE INDEX idx_secrets_user_updated_at ON secrets(user_id, updated_at);

