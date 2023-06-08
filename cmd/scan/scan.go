/*
Package scan

Copyright © 2023 MrPMillz
*/
package scan

import (
	"github.com/mr-pmillz/goforit/runner"
	"github.com/spf13/cobra"
	"log"
	"os"
	"reflect"
)

type Options struct {
	scanOptions runner.Options
}

func configureCommand(cmd *cobra.Command) {
	_ = runner.ConfigureCommand(cmd)
}

func (opts *Options) LoadFromCommand(cmd *cobra.Command) error {
	return opts.scanOptions.LoadFromCommand(cmd)
}

// Command represents the scan command
var Command = &cobra.Command{
	Use:   "scan",
	Short: "Run Nmap against a target",
	Long: `Run an Nmap scan against a target or a list of targets.

Example Commands:
	goforit scan --config config.yaml
	goforit scan -t scanme.nmap.org --output /tmp/scanme.nmap.org -v
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if configFileSet, err := cmd.Flags().GetBool("configfileset"); !configFileSet && err == nil {
			_ = cmd.MarkPersistentFlagRequired("output")
			_ = cmd.MarkPersistentFlagRequired("target")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		opts := Options{}
		if err = opts.LoadFromCommand(cmd); err != nil {
			log.Fatalf("Cloud not LoadFromCommand %+v\n", err)
		}
		switch {
		case reflect.TypeOf(opts.scanOptions.Target).Kind() == reflect.String:
			if opts.scanOptions.Target.(string) == "" {
				log.Fatalf("TARGET value cannot be empty!")
			}
		case opts.scanOptions.Output == "":
			log.Fatalf("OUTPUT config.yaml value cannot be empty.")
		}

		if err = os.MkdirAll(opts.scanOptions.Output, 0750); err != nil {
			log.Fatalf("Error creating output dir: %+v\n", err)
		}

		target, err := runner.NewTargets(&opts.scanOptions)
		if err != nil {
			log.Fatalf("Could not create new target object %+v\n", err)
		}
		if err = target.Scanner(&opts.scanOptions); err != nil {
			log.Fatalf("Error in runner.Scanner():\n%+v\n", err)
		}
	},
}

func init() {
	configureCommand(Command)
}
