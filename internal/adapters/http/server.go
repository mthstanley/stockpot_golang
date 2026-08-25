package http

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mthstanley/stockpot/internal/adapters/postgres"
	user "github.com/mthstanley/stockpot/internal/core"
)

type Server struct {
	userService user.Service
}

func NewServer(db *pgxpool.Pool) Server {
	userRepo := postgres.NewUserRepository(db)
	userService := user.NewDefaultService(userRepo)
	return Server{
		userService,
	}
}

func (s Server) Serve(addr string) error {
	router := NewRouter(s.userService)

	api := http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests
	go func() {
		log.Printf("API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a beffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return err
	case <-shutdown:
		log.Println("Start shutdown")

		// Give outstanding requests a deadline for completion.
		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Asking listener to shutdown and load shed.
		err := api.Shutdown(ctx)
		if err != nil {
			log.Printf("Graceful shutdown did not complete in %v : %s", timeout, err)
			err = api.Close()
			return err
		}
	}
	return nil
}
