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
		// Only bind the flags of the command being executed. Subcommands share
		// flag names (e.g. db-host), so binding every subcommand's flags would
		// let whichever was bound last shadow the flags actually passed in.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
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
	rootCMD.AddCommand(userCMD)
	cobra.OnInitialize(initConfig)
}

func er(msg any) {
	fmt.Println("Error:", msg)
	os.Exit(1)
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

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
