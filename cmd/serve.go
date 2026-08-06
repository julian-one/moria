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

	"moria/route"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

// shutdownGrace must stay under kubernetes' 30s termination grace period,
// after which the pod is SIGKILLed. See docs/graceful-shutdown.md.
const shutdownGrace = 15 * time.Second

var serve = &cobra.Command{
	Use:   "serve",
	Short: "Serve the moria auth API",
	PreRun: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlag("port", cmd.Flags().Lookup("port"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		srv := &http.Server{
			Addr:              ":" + viper.GetString("port"),
			Handler:           route.Initialize(route.Config{}),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       2 * time.Minute,
		}

		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			slog.Info("server listening", "addr", srv.Addr)
			// ErrServerClosed is the normal return after Shutdown, not a failure.
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
	serve.Flags().StringP("port", "p", "8081", "Port for the auth API")
}
