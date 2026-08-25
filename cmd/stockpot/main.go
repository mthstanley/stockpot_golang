package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile string

	rootCMD = &cobra.Command{
		Use:   "stockpot",
		Short: "Used to run various aspects of the stockpot application",
	}
)

// Execute executes the root command.
func main() {
	err := godotenv.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Println("No .env file detected")
		} else {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}
	rootCMD.Execute()
}

func init() {
	rootCMD.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.stockpot.yaml)")
	rootCMD.AddCommand(serverCMD)
	rootCMD.AddCommand(migrateCMD)
	cobra.OnInitialize(initConfig)
}

func er(msg any) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func bindAllCommandFlags(cmd *cobra.Command) {
	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())

	for _, subCMD := range cmd.Commands() {
		bindAllCommandFlags(subCMD)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			er(err)
		}

		// Search config in home directory with name ".stockpot" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".stockpot")
	}

	viper.SetEnvPrefix("STOCKPOT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	bindAllCommandFlags(rootCMD)

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
