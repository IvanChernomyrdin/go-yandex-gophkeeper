package cli

import (
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/crypto"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
	"github.com/spf13/cobra"
)

// для тестов
var (
	NewAPIClient       = api.NewClient
	EncryptPayload     = crypto.EncryptPayload
	ReadMasterPassword = func(cmd *cobra.Command, fromStdin bool) (string, error) {
		return readMasterPassword(cmd, fromStdin)
	}
	SaveSecretsToFile = memory.SaveToFile
	DecryptPayload    = crypto.DecryptPayload
)
