package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mthstanley/stockpot/internal/adapters/postgres"
	"github.com/mthstanley/stockpot/internal/core/auth"
	"github.com/mthstanley/stockpot/internal/core/user"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var userCMD = &cobra.Command{
	Use:   "user",
	Short: "Manage stockpot users",
}

var userCreateCMD = &cobra.Command{
	Use:   "create",
	Short: "Create a user",
	Long: `This command creates a stockpot user along with the credentials used to sign in.

If no password is provided via --password or STOCKPOT_PASSWORD, it is prompted
for interactively, or read from the first line of stdin when stdin is not a terminal.`,
	Run: runUserCreate,
}

type UserCreateConfig struct {
	Name       string `mapstructure:"name"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	DBHost     string `mapstructure:"db-host"`
	DBPort     string `mapstructure:"db-port"`
	DBUsername string `mapstructure:"db-username"`
	DBPassword string `mapstructure:"db-password"`
	DBDatabase string `mapstructure:"db-database"`
}

func init() {
	userCreateCMD.Flags().String("name", "", "The display name of the user.")
	userCreateCMD.Flags().String("username", "", "The username the user signs in with.")
	userCreateCMD.Flags().String("password", "", "The password the user signs in with (prompted for if omitted).")
	userCreateCMD.Flags().StringP("db-host", "o", "localhost", "The database host.")
	userCreateCMD.Flags().StringP("db-port", "p", "5432", "The database port.")
	userCreateCMD.Flags().StringP("db-username", "u", "postgres", "The database username.")
	userCreateCMD.Flags().StringP("db-password", "w", "postgres", "The database password.")
	userCreateCMD.Flags().StringP("db-database", "d", "stockpot", "The database database.")
	userCMD.AddCommand(userCreateCMD)
}

func readPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Password: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}

		fmt.Fprint(os.Stderr, "Confirm password: ")
		confirm, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}

		if string(password) != string(confirm) {
			return "", errors.New("passwords do not match")
		}
		return string(password), nil
	}

	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(password, "\r\n"), nil
}

func runUserCreate(cmd *cobra.Command, args []string) {
	var config UserCreateConfig
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if config.Name == "" {
		log.Fatalf("--name is required")
	}
	if config.Username == "" {
		log.Fatalf("--username is required")
	}

	if config.Password == "" {
		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}
		config.Password = password
	}
	if config.Password == "" {
		log.Fatalf("password must not be empty")
	}

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
		log.Fatalf("could not connect to db: %v", err)
	}
	defer db.Close()

	userService := user.NewDefaultService(postgres.NewUserRepository(db))
	// the jwt secret is only used for issuing and validating tokens, which
	// creating a user does not do
	authService := auth.NewDefaultService(postgres.NewAuthUserRepository(db), userService, "")

	u, err := userService.Create(ctx, user.User{Name: config.Name})
	if err != nil {
		log.Fatalf("failed to create user: %v", err)
	}

	a, err := authService.CreateAuthUser(ctx, *u, auth.UsernameAndPassword{
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		log.Fatalf("failed to create auth user: %v", err)
	}

	fmt.Printf("Created user %q (id %d) with username %q\n", a.Name, *a.ID, a.Username)
}
