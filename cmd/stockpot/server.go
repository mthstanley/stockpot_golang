package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mthstanley/stockpot/internal/adapters/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCMD = &cobra.Command{
	Use:   "server",
	Short: "Run the server",
	Long:  `This command initializes and runs the stockpot api server.`,
	Run:   runServer,
}

type ServerConfig struct {
	Addr       string `mapstructure:"addr"`
	APIDomain  string `mapstructure:"api-domain"`
	DBHost     string `mapstructure:"db-host"`
	DBPort     string `mapstructure:"db-port"`
	DBUsername string `mapstructure:"db-username"`
	DBPassword string `mapstructure:"db-password"`
	DBDatabase string `mapstructure:"db-database"`
	JWTSecret  string `mapstructure:"jwt-secret"`
}

func init() {
	serverCMD.Flags().StringP("addr", "a", "0.0.0.0:8080", "The host and port the API should bind to")
	serverCMD.Flags().StringP("api-domain", "n", "http://localhost", "The domain name of the API")
	serverCMD.Flags().StringP("db-host", "o", "localhost", "The database host.")
	serverCMD.Flags().StringP("db-port", "p", "5432", "The database port.")
	serverCMD.Flags().StringP("db-username", "u", "postgres", "The database username.")
	serverCMD.Flags().StringP("db-password", "w", "postgres", "The database password.")
	serverCMD.Flags().StringP("db-database", "d", "stockpot", "The database database.")
	serverCMD.Flags().StringP("jwt-secret", "j", "stockpot", "Secret to be used in JWT signing algorithm.")
}

func runServer(cmd *cobra.Command, args []string) {
	var config ServerConfig
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("main : Started")
	defer log.Println("main : Completed")

	dbConnString := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s",
		config.DBUsername,
		config.DBPassword,
		config.DBHost,
		config.DBPort,
		config.DBDatabase,
	)

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbConnString)
	if err != nil {
		log.Fatalf("main : could not connect to db : %v", err)
	}
	server := http.NewServer(db, config.JWTSecret, config.APIDomain)

	err = server.Serve(config.Addr)

	if err != nil {
		log.Fatalf("main : could not stop server gracefully : %v", err)
	}
}
