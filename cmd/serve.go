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
	"moria/internal/email"
	"moria/internal/logger"
	"moria/internal/session"
	"moria/route"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Moria HTTP server",
	Long:  `The serve command starts the auth API, including the cluster-internal /internal/* routes`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP("port", "p", "8081", "port for the auth API")
	serveCmd.Flags().String("db-path", "./moria.db", "path to the SQLite database")
	serveCmd.Flags().String("db-schema", "./schema/model.sql", "path to the database schema file")

	_ = viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("database.path", serveCmd.Flags().Lookup("db-path"))
	_ = viper.BindPFlag("database.schema", serveCmd.Flags().Lookup("db-schema"))
}

// shutdownGrace must stay under kubernetes' 30s termination grace period,
// after which the pod is SIGKILLed. See docs/graceful-shutdown.md.
const shutdownGrace = 15 * time.Second

func runServe(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	l := logger.New(slog.LevelInfo)
	slog.SetDefault(l)

	db, err := database.New(
		viper.GetString("database.path"),
		viper.GetString("database.schema"),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }()

	emailClient := email.New(
		viper.GetString("resend.api_key"),
		viper.GetString("resend.from_email"),
		viper.GetString("server.base_url"),
	)

	config := route.Config{
		Logger:     l,
		DB:         db,
		Email:      emailClient,
		SigningKey: viper.GetString("hmac.signing_key"),
	}

	srv := &http.Server{
		Addr:              ":" + viper.GetString("server.port"),
		Handler:           route.Initialize(config),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		// Leaves room for the synchronous Resend call in register/forgot-password.
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  2 * time.Minute,
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		l.Info("server listening", "addr", srv.Addr)
		// ErrServerClosed is the normal return after Shutdown, not a failure.
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server stopped: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		session.PurgeLoop(gctx, l, db)
		return nil
	})

	g.Go(func() error {
		<-gctx.Done()
		l.Info("shutting down", "grace", shutdownGrace)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil {
		return err
	}
	l.Info("shutdown complete")
	return nil
}
