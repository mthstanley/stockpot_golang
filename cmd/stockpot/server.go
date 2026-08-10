package main

import (
	"log"

	"github.com/mthstanley/stockpot/internal/adapters/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the server",
	Long:  `This command initializes and runs the stockpot api server.`,
	Run:   run,
}

func init() {
	serverCmd.Flags().StringP("addr", "a", "0.0.0.0:8000", "The host and port the API should bind to")
	viper.BindPFlag("addr", serverCmd.Flags().Lookup("addr"))
	viper.SetDefault("addr", "0.0.0.0:8000")
}

func run(cmd *cobra.Command, args []string) {

	// =========================================================================
	// App Starting

	log.Printf("main : Started")
	defer log.Println("main : Completed")

	// =========================================================================
	// Start API Service
	server := http.NewServer()

	err := server.Serve(viper.GetString("addr"))

	if err != nil {
		log.Fatalf("main : could not stop server gracefully : %v", err)
	}
}
