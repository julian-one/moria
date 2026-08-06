package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"moria/internal/database"
	"moria/internal/user"
)

var createUser = &cobra.Command{
	Use:   "create-user",
	Short: "Create a user",
	PreRun: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlags(cmd.Flags())
		cmd.Flags().VisitAll(func(f *pflag.Flag) { _ = viper.BindEnv(f.Name) })
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		role, err := user.ParseRole(viper.GetString("role"))
		if err != nil {
			return err
		}

		db, err := database.New(cmd.Context(), viper.GetString("database-url"))
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		u, err := user.Create(cmd.Context(), db,
			viper.GetString("username"),
			viper.GetString("email"),
			viper.GetString("password"),
			role,
		)
		if err != nil {
			return err
		}
		fmt.Println(u.UserID)
		return nil
	},
}

func init() {
	createUser.Flags().String("database-url", "", "PostgreSQL connection URL")
	createUser.Flags().String("username", "", "Username")
	createUser.Flags().String("email", "", "Email")
	createUser.Flags().String("password", "", "Password")
	createUser.Flags().String("role", "user", "Role")
}
