package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"moria/internal/database"
	"moria/route"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

const shutdownGrace = 15 * time.Second

var serve = &cobra.Command{
	Use:   "serve",
	Short: "Serve the moria auth API",
	PreRun: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlags(cmd.Flags())
		cmd.Flags().VisitAll(func(f *pflag.Flag) { _ = viper.BindEnv(f.Name) })
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		db, err := database.New(viper.GetString("database-path"))
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		if path := viper.GetString("seed-path"); path != "" {
			stmts, err := os.ReadFile(path) // #nosec G304 -- operator-supplied config

			if err != nil {
				return fmt.Errorf("failed to read seed file: %w", err)
			}
			if _, err := db.Exec(string(stmts)); err != nil {
				return fmt.Errorf("failed to apply seed file: %w", err)
			}
			slog.Warn("applied seed file", "path", path)
		}

		srv := &http.Server{
			Addr:              ":" + viper.GetString("listen-port"),
			Handler:           route.Initialize(route.Config{DB: db}),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       2 * time.Minute,
		}

		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			slog.Info("server listening", "addr", srv.Addr)
			if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("server stopped: %w", err)
			}
			return nil
		})

		g.Go(func() error {
			<-gctx.Done()
			slog.Info("shutting down", "grace", shutdownGrace)

			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		})

		if err := g.Wait(); err != nil {
			return err
		}
		slog.Info("shutdown complete")
		return nil
	},
}

func init() {
	serve.Flags().StringP("listen-port", "p", "8081", "Port for the auth API")
	serve.Flags().String("database-path", "moria.db", "Path to the SQLite database")
	serve.Flags().
		String("seed-path", "", "SQL file applied after the schema (initial-user bootstrap)")
}
