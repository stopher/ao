package cmd

import (
	"fmt"
	"strings"

	"github.com/skatteetaten/ao/pkg/auroraconfig"

	pkgEditCmd "github.com/skatteetaten/ao/pkg/editcmd"
	"github.com/spf13/cobra"
)

var editcmdObject = &pkgEditCmd.EditcmdClass{
	Configuration: config,
}

var editCmd = &cobra.Command{
	Use:   "edit [env/]file",
	Short: "Edit a single file in the AuroraConfig repository, or a secret in a vault",
	Long: `Edit a single file in the AuroraConfig repository, or a secret in a vault.
The file can be specified using unique shortened name, so given that the file superapp-test/about.json exists, then the command

	ao edit test/about

will edit this file, if there is no other file matching the same shortening.`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 1 {
			cmd.Usage()
			return
		}

		if output, err := editcmdObject.FuzzyEditFile(args); err == nil {
			if output != "" {
				fmt.Println(output)
			}
			auroraconfig.UpdateLocalRepository(config.GetAffiliation(), aoConfig)
		} else {
			fmt.Println(err)
		}
	},
}

var editFileCmd = &cobra.Command{
	Use:   "file [env/]<filename>",
	Short: "Edit a single configuration file",
	Long: `Edit a single configuration file or a secret in a vault.
The file can be specified using unique shortened name, so given that the file superapp-test/about.json exists, then the command

	ao edit test/about

will edit this file, if there is no other file matching the same shortening.`,
	Annotations: map[string]string{
		CallbackAnnotation: "GetFiles",
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Usage()
			return
		}

		if output, err := editcmdObject.FuzzyEditFile(args); err == nil {
			if output != "" {
				fmt.Println(output)
			}
			auroraconfig.UpdateLocalRepository(config.GetAffiliation(), aoConfig)
		} else {
			fmt.Println(err)
		}
	},
}

var editVaultCmd = &cobra.Command{
	Use:   "vault <vaultname> | <vaultname>/<secretname> | <vaultname> <secretname>",
	Short: "Edit a vault or a secret in a vault",
	Long: `This command will edit the content of the given vault.
The editor will present a JSON view of the vault.
The secrets will be presented as Base64 encoded strings.
If secret-name is given, the editor will present the decoded secret string for editing.`,
	Run: func(cmd *cobra.Command, args []string) {
		var vaultname string
		var secretname string
		var output string
		var err error
		if len(args) == 1 {
			if strings.Contains(args[0], "/") {
				parts := strings.Split(args[0], "/")
				vaultname = parts[0]
				secretname = parts[1]
			} else {
				vaultname = args[0]
			}
		} else if len(args) == 2 {
			vaultname = args[0]
			secretname = args[1]
		} else {
			cmd.Usage()
			return
		}

		if secretname != "" {
			output, err = editcmdObject.EditSecret(vaultname, secretname)
		} else {
			output, err = editcmdObject.EditVault(vaultname)
		}
		if err == nil {
			fmt.Print(output)
		} else {
			fmt.Println(err)
		}

	},
}

func init() {
	RootCmd.AddCommand(editCmd)
	editCmd.AddCommand(editFileCmd)
	editCmd.AddCommand(editVaultCmd)
}
