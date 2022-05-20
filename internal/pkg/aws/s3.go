package aws

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	sConfig "github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

const (
	backupPrefix = "backups"
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
		BackupFileKey: fmt.Sprintf("%s/%s", backupPrefix, serverConfig.AppName),
		TmpWritePath:  fmt.Sprintf("/tmp/%s", serverConfig.AppName),
	}, nil
}

func (c *Client) UploadBackupFile(ctx context.Context) error {
	file, err := os.Open(c.TmpWritePath)
	if err != nil {
		return err
	}

	uploader := manager.NewUploader(c.S3)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(c.BackupFileKey),
		Body:   file,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) downloadFileExists(ctx context.Context) (bool, error) {
	paginator := s3.NewListObjectsV2Paginator(c.S3, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.Bucket),
		Prefix: aws.String(backupPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return false, err
		}
		for _, obj := range page.Contents {
			if *obj.Key == c.BackupFileKey {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Client) DownloadOrCreateBackupFile(ctx context.Context) error {
	tmpFile, err := os.Create(c.TmpWritePath)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	exists, err := c.downloadFileExists(context.Background())
	if exists {
		downloader := manager.NewDownloader(c.S3)
		_, err = downloader.Download(ctx, tmpFile, &s3.GetObjectInput{
			Bucket: aws.String(c.Bucket),
			Key:    aws.String(c.BackupFileKey),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) WriteBackupFile(statuses []sConfig.SensorStatus) error {
	file, err := os.OpenFile(c.TmpWritePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	datawriter := bufio.NewWriter(file)
	for _, data := range statuses {
		j, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, err = datawriter.WriteString(fmt.Sprintf("%s\n", string(j)))
		if err != nil {
			return err
		}
	}
	datawriter.Flush()
	defer file.Close()
	return nil
}
