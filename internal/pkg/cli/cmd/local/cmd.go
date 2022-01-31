package local

import (
	"github.com/spf13/cobra"

	"github.com/keboola/keboola-as-code/internal/pkg/cli/cmd/local/template"
	"github.com/keboola/keboola-as-code/internal/pkg/cli/dependencies"
	"github.com/keboola/keboola-as-code/internal/pkg/cli/helpmsg"
	"github.com/keboola/keboola-as-code/internal/pkg/env"
)

func Commands(d dependencies.Provider, envs *env.Map) *cobra.Command {
	cmd := &cobra.Command{
		Use:  `local`,
		Long: helpmsg.Read(`local/long`),
	}
	cmd.AddCommand(
		ValidateCommand(d),
		PersistCommand(d),
		CreateCommand(d),
		EncryptCommand(d),
		FixPathsCommand(d),
	)

	if envs.Get(`KBC_TEMPLATES_PRIVATE_BETA`) == `true` {
		cmd.AddCommand(template.Commands(d))
	}

	return cmd
}
