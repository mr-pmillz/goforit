/*
Package cmd

Copyright © 2023 MrPMillz
*/
package cmd

import (
	"fmt"
	"github.com/mr-pmillz/goforit/cmd/scan"
	"github.com/mr-pmillz/goforit/utils"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfgFile       string
	version       = "v0.0.1"
	configFileSet bool
)

const (
	defaultConfigFileName = "config"
	envPrefix             = "GOFORIT" // CHANGEME
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "goforit", // CHANGEME
	Version: version,
	Short:   "Boilerplate skeleton Cobra Command Line Quick Starter",
	Long:    `Boilerplate skeleton Cobra Command Line Quick Starter`,
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file default location for viper to look is ~/.config/goforit/config.yaml")
	RootCmd.PersistentFlags().BoolVarP(&configFileSet, "configfileset", "", false, "Used internally by goforit to check if required args are set with and without configuration file, Do not use this flag...")
	RootCmd.AddCommand(scan.Command)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		absConfigFilePath, err := utils.ResolveAbsPath(cfgFile)
		if err != nil {
			_ = fmt.Errorf("couldn't resolve path of config file: %w", err)
			return
		}
		viper.SetConfigFile(absConfigFilePath)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Panic("Could not get user homedir. Error: %+v\n", err)
		}
		// Search config in $HOME/.config/goforit/config.yaml directory with name "config.yaml"
		viper.AddConfigPath(fmt.Sprintf("%s/.config/goforit", homeDir))
		viper.SetConfigType("yaml")
		viper.SetConfigName(defaultConfigFileName)
	}

	// If a config file is found, read it.
	if err := viper.ReadInConfig(); err == nil {
		configFileSet = true
		fmt.Printf("Using config file: %v", viper.ConfigFileUsed())
	}
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv() // read in environment variables that match
	bindFlags(RootCmd)
}

// bindFlags Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		err := viper.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		if err != nil {
			return
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				return
			}
		}
	})
}
