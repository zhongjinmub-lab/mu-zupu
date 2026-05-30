package storage

import (
	"context"
	"errors"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Bucket     string
	UseSSL     bool
	PublicBase string
}

type Client struct {
	MinIO *minio.Client
	Config
}

func NewMinIO(cfg Config) (Client, error) {
	if cfg.Endpoint == "" {
		return Client{}, errors.New("storage endpoint is required")
	}
	if cfg.AccessKey == "" {
		return Client{}, errors.New("storage access key is required")
	}
	if cfg.SecretKey == "" {
		return Client{}, errors.New("storage secret key is required")
	}
	if cfg.Bucket == "" {
		return Client{}, errors.New("storage bucket is required")
	}
	c, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return Client{}, err
	}
	return Client{MinIO: c, Config: cfg}, nil
}

func (c Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.MinIO.BucketExists(ctx, c.Bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.MinIO.MakeBucket(ctx, c.Bucket, minio.MakeBucketOptions{})
}

func (c Client) Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	_, err := c.MinIO.PutObject(ctx, c.Bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (c Client) Get(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	return c.MinIO.GetObject(ctx, c.Bucket, objectKey, minio.GetObjectOptions{})
}
