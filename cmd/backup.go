package cmd

import (
	"fmt"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/aws"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/postgres"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		serverConfig := config.ServerConfig{
			AppName:     viper.GetString("APP_NAME"),
			PostgresURL: viper.GetString("DATABASE_URL"),
			S3Config: config.S3Config{
				AccessKeyID:      viper.GetString("BUCKETEER_AWS_ACCESS_KEY_ID"),
				SecretAccessKey:  viper.GetString("BUCKETEER_AWS_SECRET_ACCESS_KEY"),
				Region:           viper.GetString("BUCKETEER_AWS_REGION"),
				Bucket:           viper.GetString("BUCKETEER_BUCKET_NAME"),
				RetentionEnabled: viper.GetBool("DB_RETENTION_ENABLED"),
				MaxRetentionRows: parseRetentionRowsConfig(viper.GetString("DB_MAX_RETENTION_ROWS")),
			},
		}

		fmt.Println("App name:", serverConfig.AppName)

		postgresClient, err := postgres.NewPostgresClient(serverConfig.PostgresURL)
		if err != nil {
			panic(err)
		}

		count, err := postgresClient.GetRowCount()
		if err != nil {
			panic(err)
		}

		fmt.Println("Row count:", count)

		rows, err := postgresClient.GetAllRows()
		if err != nil {
			panic(err)
		}

		awsClient, err := aws.NewClient(serverConfig)
		if err != nil {
			panic(err)
		}

		append := false
		err = awsClient.WriteBackupFile(rows, append)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	// rootCmd.AddCommand(backupCmd)
}
