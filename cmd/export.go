// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/skatteetaten/ao/pkg/exportcmd"
	"github.com/spf13/cobra"
)

var outputFolder string

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export files | file [env/]<filename> | | vaults | vault <vaultname> | secret <vaultname> <secretname> | adc",
	Short: "Exports auroraconf, vaults or secrets to one or more files",
	Long: `Exports the entire affiliation or a specific file to a set of files.
The files will be printed to standard out, but can also be stored individually in a folder structure
by using the output-folder option.`,
	Run: func(cmd *cobra.Command, args []string) {
		exportcmdObject := &exportcmd.ExportcmdClass{
			Configuration: config,
		}
		output, err := exportcmdObject.ExportObject(args, &persistentOptions, "json", outputFolder)
		if err != nil {
			l := log.New(os.Stderr, "", 0)
			l.Println(err.Error())
			os.Exit(-1)
		} else {
			if output != "" {
				fmt.Println(output)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(exportCmd)
	exportCmd.Hidden = true

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// exportCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// exportCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	exportCmd.Flags().StringVarP(&outputFolder, "output-folder",
		"f", "", "Folder to write the export files to")
}
