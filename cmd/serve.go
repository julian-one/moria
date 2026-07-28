package cmd

import (
	"fmt"
	"log/slog"
	"net/http"

	"moria/internal/database"
	"moria/internal/email"
	"moria/internal/logger"
	"moria/route"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Moria HTTP servers",
	Long:  `The serve command starts the public auth API and the cluster-internal validation listener`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP("port", "p", "8081", "port for the public API")
	serveCmd.Flags().String("internal-port", "8082", "port for the internal validation API")
	serveCmd.Flags().String("db-path", "./moria.db", "path to the SQLite database")
	serveCmd.Flags().String("db-schema", "./schema/model.sql", "path to the database schema file")

	_ = viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("internal.port", serveCmd.Flags().Lookup("internal-port"))
	_ = viper.BindPFlag("database.path", serveCmd.Flags().Lookup("db-path"))
	_ = viper.BindPFlag("database.schema", serveCmd.Flags().Lookup("db-schema"))
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize logger
	l := logger.New(slog.LevelInfo)
	slog.SetDefault(l)

	// Initialize database
	db, err := database.New(
		viper.GetString("database.path"),
		viper.GetString("database.schema"),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize email client
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

	publicHandler := route.Initialize(ctx, config)
	internalHandler := route.InitializeInternal(ctx, config)

	port := viper.GetString("server.port")
	internalPort := viper.GetString("internal.port")

	errCh := make(chan error, 2)

	go func() {
		l.Info("public server listening", "port", port)
		errCh <- fmt.Errorf("public server stopped: %w",
			http.ListenAndServe(":"+port, publicHandler))
	}()
	go func() {
		l.Info("internal server listening", "port", internalPort)
		errCh <- fmt.Errorf("internal server stopped: %w",
			http.ListenAndServe(":"+internalPort, internalHandler))
	}()

	return <-errCh
}
