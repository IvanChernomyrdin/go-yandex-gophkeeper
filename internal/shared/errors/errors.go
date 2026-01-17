// Package errors содержит общие доменные ошибки приложения
// и утилиты для error wrapping.
//
// Эти ошибки используются в service и repository слоях
// и маппятся на HTTP-статусы в api слое.
package errors

import "errors"

var (
	// Входные данные невалидны (пустые поля, неправильный формат и т.п.)
	ErrInvalidInput = errors.New("invalid input")
	// Неверные учётные данные
	ErrInvalidCredentials = errors.New("invalid credentials")
	// Получена непредвиденная ошибка
	ErrInternal = errors.New("internal error")
	// Полученные JSON данные с ошибками
	ErrBadJSON = errors.New("bad json")
	// Неавторизован
	ErrUnauthorized = errors.New("unauthorized")
	// Ресурс уже существует (например email уже занят)
	ErrAlreadyExists = errors.New("already exists")
	// Ресурс не найден
	ErrNotFound = errors.New("not found")
	// конфликт версий(к примеру при обновлении в бд)
	ErrConflict = errors.New("conflict")
	// ожидаемая ошибка
	ErrExpectedError = errors.New("expected error")
	// неожидаемая ошибка
	ErrUnexpectedError = errors.New("unexpected error")
)

// только для секретов
var (
	// secrets
	ErrPayloadTooLarge = errors.New("payload too large")
	ErrMetaTooLarge    = errors.New("meta too large")
	ErrUserIDEmpty     = errors.New("user id cannot be empty")
)
