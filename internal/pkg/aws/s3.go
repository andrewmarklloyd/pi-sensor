package aws

import (
	"context"
	"fmt"
	"os"

	sConfig "github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

type Client struct {
	S3            *s3.Client
	Bucket        string
	BackupFileKey string
	TmpWritePath  string
}

func NewClient(serverConfig sConfig.ServerConfig) (Client, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(serverConfig.S3Config.AccessKeyID, serverConfig.S3Config.SecretAccessKey, "")),
	)
	cfg.Region = serverConfig.S3Config.Region
	if err != nil {
		return Client{}, err
	}

	client := s3.NewFromConfig(cfg)

	return Client{
		S3:            client,
		Bucket:        serverConfig.S3Config.Bucket,
		BackupFileKey: fmt.Sprintf("backups/%s", serverConfig.AppName),
		TmpWritePath:  fmt.Sprintf("/tmp/%s", serverConfig.AppName),
	}, nil
}

func (c *Client) UploadBackupFile(ctx context.Context) error {
	file, err := os.Open(c.TmpWritePath)
	if err != nil {
		return err
	}

	uploader := manager.NewUploader(c.S3)
	result, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(c.BackupFileKey),
		Body:   file,
	})
	if err != nil {
		return err
	}

	fmt.Println(result)

	return nil
}

func (c *Client) DownloadBackupFile(ctx context.Context) error {

	// downloader := manager.NewDownloader(c.S3)

	return nil
}
