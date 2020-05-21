// Copyright 2018 The CUE Authors
//
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
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"

	"cuelang.org/go/cue/lsp"
)

var configFile string

// newDefCmd creates a new eval command
func newLspCmd(c *Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "start language server",
		Long: `lsp starts the CUE language server.

At the moment, communication is limited to stdio.
`,
		RunE: mkRunE(c, runLsp),
	}

	cmd.Flags().StringVar(&configFile, "config-file", "cue-lsp.yml", "Path to yml config file.")
	// TODO: Option to include comments in output.
	return cmd
}

func runLsp(cmd *Command, args []string) error {
	err := cmd.ParseFlags(args)
	if err != nil {
		log.Fatalf("Error when parsing commands: %v", err)
	}
	//log.Infof("Starting lsp server with config: %s", configFile)
	//fmt.Fprintf(cmd.OutOrStdout(), "Starting lsp server with config: %s", configFile)

	lsp.StartLSPServer(configFile)

	return nil
}
