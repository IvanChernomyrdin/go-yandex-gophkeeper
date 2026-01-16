package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCmd создаёт CLI-команду для отображения информации о сборке.
//
// Команда выводит версию приложения и дату сборки, переданные при компиляции.
// Используется для проверки установленной версии клиента.
//
// Пример использования:
//
//	gophkeeper version
func NewVersionCmd(buildVersion, buildDate string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию и дату сборки",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"version=%s\nbuild_date=%s\n",
				buildVersion,
				buildDate,
			)
		},
	}
}
