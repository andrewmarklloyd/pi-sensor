package cmd

import (
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "pi-sensor-server",
	Short: "Server for the pi-sensor application",
	Long:  `Server for the pi-sensor application`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
	viper.SetConfigFile(".env")
}

func initConfig() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.ReadInConfig()
}
