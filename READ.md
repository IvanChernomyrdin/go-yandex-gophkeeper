# GophKeeper

Клиент-серверное приложение для безопасного хранения паролей и секретных данных.  
Данные шифруются на вашем компьютере — сервер хранит только зашифрованный набор символов.

## Функции

- Регистрация и вход
- Шифрование данных на клиенте
- Синхронизация с сервером
- Типы секретов:
  - Логины и пароли
  - Текстовые заметки
  - Бинарные данные
  - Банковские карты
  - OTP-коды


# Что нужно для запуска программы:
Создайте в корне папку certs и положите в эту папку файлы с названиями: server.crt и server.key
Создайте файл .env котоырй должен содержать переменные окружения:
  `DB_LOGIN`
  `DB_PASSWORD`
  `DB_HOST`
  `DB_PORT`
  `DB_DATABASE`
  `JWT_SIGNING_KEY`

## Основные команды (CLI)

- `gophkeeper sync` — синхронизация с сервером  
- `gophkeeper get` — показать все секреты  
- `gophkeeper get <id>` — показать секрет по ID  
- `gophkeeper set --type <тип> --title "Название" --payload '{"данные":"в json"}'` — создать новый секрет  
- `gophkeeper update <id> ...` — обновить секрет (заменяются только переданные поля)  
- `gophkeeper delete <id>` — удалить секрет  


## Быстрый запуск (2 окна терминала)

Нужно открыть **два окна терминала**:  
в первом — сервер, во втором — клиент (агент).

----------------------Важно: сервер и клиент запускаются **как обычные файлы**----------------
----------------------Команда `go run` для этого НЕ нужна-------------------------------------


### 1. Запуск сервера (первое окно)

Серверные файлы лежат в папке: `cmd/server/build/`

Запустите файл под вашу систему:

- **Windows (обычный ПК / Intel / AMD):**  
  `cmd\server\build\server_windows_amd64.exe`

- **Windows на ARM (редко):**  
  `cmd\server\build\server_windows_arm64.exe`

- **Linux (обычный ПК / сервер):**  
  `cmd/server/build/server_linux_amd64`

- **Linux на ARM (например Raspberry Pi):**  
  `cmd/server/build/server_linux_arm64`

- **macOS (Intel):**  
  `cmd/server/build/server_darwin_amd64`

- **macOS (Apple Silicon M1/M2/M3):**  
  `cmd/server/build/server_darwin_arm64`

Это окно **не закрывайте** — сервер должен работать.

### 2. Запуск клиента (второе окно)

Клиентские файлы лежат в папке: `cmd/gophkeeper/build/`

Запустите файл под вашу систему:

- **Windows (обычный ПК / Intel / AMD):**  
  `cmd\gophkeeper\build\gophkeeper_windows_amd64.exe`

- **Windows на ARM (редко):**  
  `cmd\gophkeeper\build\gophkeeper_windows_arm64.exe`

- **Linux (обычный ПК / сервер):**  
  `cmd/gophkeeper/build/gophkeeper_linux_amd64`

- **Linux на ARM:**  
  `cmd/gophkeeper/build/gophkeeper_linux_arm64`

- **macOS (Intel):**  
  `cmd/gophkeeper/build/gophkeeper_darwin_amd64`

- **macOS (Apple Silicon M1/M2/M3):**  
  `cmd/gophkeeper/build/gophkeeper_darwin_arm64`


## Запуск всех юнит тестов:
  `go test ./...`