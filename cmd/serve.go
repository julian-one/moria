package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"moria/internal/database"
	"moria/route"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:          "serve",
	Short:        "Serve the moria API",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		db, err := database.New(ctx, viper.GetString("database-url"))
		if err != nil {
			return err
		}
		defer db.Close()

		srv := &http.Server{
			Addr:    ":" + viper.GetString("port"),
			Handler: route.Initialize(route.Config{DB: db}),
		}

		serveErr := make(chan error, 1)
		go func() {
			slog.Info("server listening", "addr", srv.Addr)
			serveErr <- srv.ListenAndServe()
		}()

		select {
		case err := <-serveErr:
			return err
		case <-ctx.Done():
		}
		stop()

		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", "8080", "Port for the auth API")
	serveCmd.Flags().String("database-url", "", "PostgreSQL connection URL")
}
