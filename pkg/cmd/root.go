// Package cmd contains the jx-slack CLI commands.
package cmd

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/spf13/cobra"
)

// SlackAppOptions contains options for the root slack app command.
type SlackAppOptions struct {
	Cmd  *cobra.Command
	Args []string
}

// NewCmdRoot creates the root cobra command for jx-slack.
func NewCmdRoot() *cobra.Command {
	var options = &SlackAppOptions{}

	var rootCmd = &cobra.Command{
		Use:   "slack",
		Short: "The jenkins-x App for Slack allows you to reports pipelines and review requests in Slack",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	rootCmd.AddCommand(NewCmdRun())
	return rootCmd
}

// Run executes the root command, showing help.
func (o *SlackAppOptions) Run() error {
	return o.Cmd.Help()
}
