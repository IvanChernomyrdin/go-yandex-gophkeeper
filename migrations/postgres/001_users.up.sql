-- Создание таблицы пользователей используем email вместо login
-- т.к. он более удобен и по email можно восстанавливать доступ и его не забудут
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Индекс по email
CREATE INDEX idx_users_email ON users(email);