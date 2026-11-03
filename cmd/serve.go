package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/julian-one/moria/internal/database"
)

const (
	defaultPort       = 8000
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

func serveCmd() *cobra.Command {
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Serve the moria API",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			db, err := database.Open(ctx, viper.GetString("database-url"))
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			addr := fmt.Sprintf(":%d", viper.GetInt("port"))
			server := &http.Server{
				Addr:              addr,
				ReadHeaderTimeout: readHeaderTimeout,
				ReadTimeout:       readTimeout,
				WriteTimeout:      writeTimeout,
				IdleTimeout:       idleTimeout,
			}
			if err := server.ListenAndServe(); err != nil {
				return fmt.Errorf("serve on %s: %w", addr, err)
			}
			return nil
		},
	}
	serve.Flags().String("database-url", "", "PostgreSQL connection URL")
	serve.Flags().Int("port", defaultPort, "port to listen on")
	return serve
}
