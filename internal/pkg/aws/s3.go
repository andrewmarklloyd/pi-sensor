package aws

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	sConfig "github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	backupPrefix = "backups"
)

type Client struct {
	S3                     *s3.Client
	Bucket                 string
	RetentionBackupFileKey string
	RetentionTmpWritePath  string
	FullBackupFileKey      string
	FullBackupTmpWritePath string
}

type BucketInfo struct {
	NumVersions      int
	NumDeleteMarkers int
	Size             int64
}

func NewClient(serverConfig sConfig.ServerConfig) (Client, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(serverConfig.S3Config.AccessKeyID, serverConfig.S3Config.SecretAccessKey, "")),
	)
	if err != nil {
		return Client{}, fmt.Errorf("loading default config: %s", err)
	}

	cfg.Region = serverConfig.S3Config.Region
	cfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           serverConfig.S3Config.URL,
			SigningRegion: serverConfig.S3Config.Region,
		}, nil
	})

	client := s3.NewFromConfig(cfg)

	return Client{
		S3:                     client,
		Bucket:                 serverConfig.S3Config.Bucket,
		RetentionBackupFileKey: fmt.Sprintf("%s/%s-retention.json", backupPrefix, serverConfig.AppName),
		RetentionTmpWritePath:  fmt.Sprintf("/tmp/%s-retention.json", serverConfig.AppName),
		FullBackupFileKey:      fmt.Sprintf("%s/%s-full-backup.json", backupPrefix, serverConfig.AppName),
		FullBackupTmpWritePath: fmt.Sprintf("/tmp/%s-full-backup.json", serverConfig.AppName),
	}, nil
}

func (c *Client) UploadBackupFile(ctx context.Context, tmpWritePath, backupFileKey string) error {
	file, err := os.Open(tmpWritePath)
	if err != nil {
		return fmt.Errorf("opening tmp file: %s", err)
	}

	uploader := manager.NewUploader(c.S3)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(backupFileKey),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("uploading backup file to s3: %s", err)
	}

	return nil
}

func (c *Client) backupFileExistsInS3(ctx context.Context, key string) (bool, error) {
	paginator := s3.NewListObjectsV2Paginator(c.S3, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.Bucket),
		Prefix: aws.String(backupPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return false, fmt.Errorf("paginating response from AWS: %s", err)
		}
		for _, obj := range page.Contents {
			if *obj.Key == key {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Client) DownloadOrCreateBackupFile(ctx context.Context, tmpWritePath, backupFileKey string) error {
	// os.Create truncates the file if it exists
	tmpFile, err := os.Create(tmpWritePath)
	if err != nil {
		return fmt.Errorf("creating tmp file: %s", err)
	}
	defer tmpFile.Close()

	exists, err := c.backupFileExistsInS3(ctx, backupFileKey)
	if err != nil {
		return fmt.Errorf("checking if backup file exists in s3: %w", err)
	}
	if exists {
		downloader := manager.NewDownloader(c.S3)
		_, err = downloader.Download(ctx, tmpFile, &s3.GetObjectInput{
			Bucket: aws.String(c.Bucket),
			Key:    aws.String(backupFileKey),
		})
		if err != nil {
			return fmt.Errorf("downloading backup file from S3: %s", err)
		}
	}

	return nil
}

func (c *Client) WriteBackupFile(statuses []sConfig.SensorStatus, append bool, tmpWritePath string) error {
	var mode int
	if append {
		mode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		mode = os.O_CREATE | os.O_WRONLY
	}

	file, err := os.OpenFile(tmpWritePath, mode, 0644)
	if err != nil {
		return fmt.Errorf("opening tmp file: %s", err)
	}

	datawriter := bufio.NewWriter(file)
	for _, data := range statuses {
		j, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshalling sensor status: %s", err)
		}
		_, err = datawriter.WriteString(fmt.Sprintf("%s\n", string(j)))
		if err != nil {
			return fmt.Errorf("writing string to datawriter: %s", err)
		}
	}
	datawriter.Flush()
	defer file.Close()
	return nil
}

func (c *Client) GetBucketInfo(ctx context.Context) (BucketInfo, error) {
	input := &s3.ListObjectVersionsInput{
		Bucket: aws.String(c.Bucket),
	}

	versionOut, err := c.S3.ListObjectVersions(ctx, input)
	if err != nil {
		return BucketInfo{}, fmt.Errorf("listing object versions: %w", err)
	}

	objectOut, err := c.S3.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(c.Bucket),
	})

	if err != nil {
		return BucketInfo{}, fmt.Errorf("cannot ListObjectsV2 in %s/%s: %s", c.Bucket, backupPrefix, err.Error())
	}

	var size int64
	for _, object := range objectOut.Contents {
		size += *object.Size
	}

	return BucketInfo{
		NumVersions:      len(versionOut.Versions),
		NumDeleteMarkers: len(versionOut.DeleteMarkers),
		Size:             size,
	}, nil
}
