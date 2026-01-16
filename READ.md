gophkeeper:
    - получает команду, её парсит и выполняет
    - хранит access/refresh токены локально в ~/.gophkeeper/config.json (права 0600), при refresh получает данные из поля refresh и обновляет access;



Структура проекта представлена ниже.

gophkeeper/
├── .github/
│   └── workflows/
│   
├── cmd/
│   ├── server/
│   │   └── main.go
│   │       # Точка входа сервера:
│   │       # - загрузка конфигурации
│   │       # - инициализация БД и миграций
│   │       # - инициализация сервисов
│   │       # - регистрация HTTP-роутов
│   │       # - запуск HTTP-сервера
│   │
│   └── agent/
│       └── main.go
│           # Точка входа CLI-клиента:
│           # - инициализация CLI
│           # - регистрация команд
│           # - запуск приложения
│
├── internal/
│   ├── server/
│   │   ├── api/
│   │   │   ├── auth.go
│   │   │   │   # HTTP-хендлеры:
│   │   │   │   # - регистрация
│   │   │   │   # - логин
│   │   │   │   # - refresh токенов
│   │   │   │
│   │   │   └── secrets.go
│   │   │       # HTTP CRUD-хендлеры для приватных данных
│   │   │
│   │   ├── service/
│   │   │   ├── auth.go
│   │   │   │   # Бизнес-логика:
│   │   │   │   # - регистрация
│   │   │   │   # - проверка пароля
│   │   │   │   # - генерация JWT
│   │   │   │
│   │   │   └── secrets.go
│   │   │       # Бизнес-логика:
│   │   │       # - хранение секретов
│   │   │       # - синхронизация
│   │   │       # - optimistic locking (version)
│   │   │       # - разрешение конфликтов
│   │   │
│   │   ├── repository/
│   │   │   ├── users.go
│   │   │   │   # Доступ к таблице users (PostgreSQL)
│   │   │   │
│   │   │   └── secrets.go
│   │   │       # Доступ к таблице secrets (PostgreSQL)
│   │   │
│   │   ├── models/
│   │   │   ├── user.go
│   │   │   │   # Доменные модели пользователя
│   │   │   │
│   │   │   └── secret.go
│   │   │       # Доменные модели секретных данных
│   │   │       # (не зависят от HTTP и DTO)
│   │   │
│   │   ├── crypto/
│   │   │   ├── password.go
│   │   │   │   # Хэширование и проверка паролей
│   │   │   │
│   │   │   └── jwt.go
│   │   │       # Генерация и валидация JWT
│   │   │
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   │   # Проверка JWT / access токенов
│   │   │   │
│   │   │   └── logger.go
│   │   │       # HTTP middleware для логирования запросов
│   │   │
│   │   └── config/
│   │       └── config.go
│   │           # Загрузка и валидация конфигурации сервера
│   │
│   ├── agent/
│   │   ├── cli/
│   │   │   ├── root.go
│   │   │   │   # Корневая CLI-команда + help
│   │   │   │
│   │   │   ├── version.go
│   │   │   │   # Команда version:
│   │   │   │   # - версия
│   │   │   │   # - дата сборки
│   │   │   │
│   │   │   ├── auth.go
│   │   │   │   # CLI-команды:
│   │   │   │   # - register
│   │   │   │   # - login
│   │   │   │   # - logout
│   │   │   │
│   │   │   └── secrets.go
│   │   │       # CLI-команды:
│   │   │       # - add
│   │   │       # - list
│   │   │       # - get
│   │   │       # - update
│   │   │       # - delete
│   │   │
│   │   ├── api/
│   │   │   ├── client.go
│   │   │   │   # HTTP-клиент (JWT, retry, refresh)
│   │   │   │
│   │   │   ├── auth.go
│   │   │   │   # Вызовы auth API сервера
│   │   │   │
│   │   │   └── secrets.go
│   │   │       # Вызовы CRUD API сервера
│   │   │
│   │   ├── config/
│   │   │   └── config.go
│   │   │       # Конфигурация CLI:
│   │   │       # - URL сервера
│   │   │       # - токены
│   │   │
│   │   └── crypto/
│   │       ├── kdf.go
│   │       │   # Derivation мастер-ключа
│   │       │
│   │       ├── encrypt.go
│   │       │   # Шифрование payload
│   │       │
│   │       └── decrypt.go
│   │           # Дешифрование payload
│   │           # Ключи НИКОГДА не передаются серверу
│   │
│   └── shared/
│       ├── logger/
│       │   └── logging.go
│       │       # Общий логгер (server + agent)
│       │
│       ├── models/
│       │   └── models.go
│       │       # Общие enum-типы (SecretType и т.д.)
│       │
│       ├── errors/
│       │   └── errors.go
│       │       # Общие ошибки и error wrapping
│       │
│       └── utils/
│           └── utils.go
│               # Общие утилиты
│
├── migrations/
│   ├── 001_users.up.sql
│   ├── 001_users.down.sql
│   ├── 002_secrets.up.sql
│   └── 002_secrets.down.sql
│       # Миграции БД (PostgreSQL)
│
├── configs/
│   ├── server.yaml
│   │   # Конфигурация сервера
│   └── agent.yaml
│       # Конфигурация клиента
│
├── api/
│   └── openapi.yaml
│       # Swagger / OpenAPI спецификация API
│
├── go.mod
├── go.sum
└── README.md
    # Архитектура, безопасность, синхронизация, запуск