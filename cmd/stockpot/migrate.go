package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	migrations "github.com/mthstanley/stockpot"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var migrateCMD = &cobra.Command{
	Use:   "migrate",
	Short: "Run migrations against DB",
	Long:  `This command migrates the DB to the latest schema.`,
	Run:   runMigrate,
}

type MigrateConfig struct {
	DBHost     string `mapstructure:"db-host"`
	DBPort     string `mapstructure:"db-port"`
	DBUsername string `mapstructure:"db-username"`
	DBPassword string `mapstructure:"db-password"`
	DBDatabase string `mapstructure:"db-database"`
}

func init() {
	migrateCMD.Flags().StringP("db-host", "o", "localhost", "The database host.")
	migrateCMD.Flags().StringP("db-port", "p", "5432", "The database port.")
	migrateCMD.Flags().StringP("db-username", "u", "postgres", "The database username.")
	migrateCMD.Flags().StringP("db-password", "w", "postgres", "The database password.")
	migrateCMD.Flags().StringP("db-database", "d", "stockpot", "The database database.")
}

func runMigrate(cmd *cobra.Command, args []string) {
	var config MigrateConfig
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbConnString := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s",
		config.DBUsername,
		config.DBPassword,
		config.DBHost,
		config.DBPort,
		config.DBDatabase,
	)

	pool, err := pgxpool.New(context.Background(), dbConnString)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(migrations.Embedded)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := goose.Up(db, "db/migrations"); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
}
