package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:     "moria",
	Version: "0.1.0",
	Short:   "Moria - The authentication service",
	Long: `Moria is the standalone authentication service.
		It owns users and sessions, and serves registration, login,
		password recovery, and session validation.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is ./config.json)")

	if err := viper.BindEnv("resend.api_key", "RESEND_API_KEY"); err != nil {
		slog.Error("failed to bind env", "key", "resend.api_key", "error", err)
	}
	if err := viper.BindEnv("hmac.signing_key", "HMAC_SIGNING_KEY"); err != nil {
		slog.Error("failed to bind env", "key", "hmac.signing_key", "error", err)
	}
}

func initializeConfig() error {
	viper.SetDefault("server.port", "8081")
	viper.SetDefault("server.base_url", "https://julian-one.com")
	viper.SetDefault("internal.port", "8082")
	viper.SetDefault("database.path", "./moria.db")
	viper.SetDefault("database.schema", "./schema/model.sql")
	viper.SetDefault("resend.from_email", "noreply@contact.julian-one.com")
	viper.SetDefault("resend.api_key", "")
	viper.SetDefault("hmac.signing_key", "")

	viper.SetEnvPrefix("MORIA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			slog.Warn("no config file found, using defaults and environment")
		} else {
			return fmt.Errorf("reading config file: %w", err)
		}
	}

	return nil
}
