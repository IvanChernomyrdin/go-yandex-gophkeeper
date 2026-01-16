// Package main содержит точку входа клиентского CLI-приложения.
//
// Пакет отвечает за запуск консольного клиента и передачу информации о версии и дате сборки в CLI-слой приложения.
package main

import "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"

var (
	// buildVersion содержит версию приложения, передаваемую при сборке.
	// По умолчанию используется значение "dev".
	buildVersion = "dev"
	// buildDate содержит дату сборки приложения.
	// По умолчанию используется значение "unknown".
	buildDate = "unknown"
)

func main() {
	cli.Execute(buildVersion, buildDate)
}
