package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
)

type Storage struct {
	client *minio.Client
}

func NewStorage(client *minio.Client) (*Storage, error) {
	return &Storage{client: client}, nil
}

func (s *Storage) PresignedUpload(ctx context.Context, videoID, ext string, ttl time.Duration) (string, error) {
	key := fmt.Sprintf("%s/source%s", videoID, ext)
	u, err := s.client.PresignedPutObject(ctx, "raw-input", key, ttl)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *Storage) PresignedDownload(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, bucket, key, ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *Storage) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Storage) EnsureBuckets(ctx context.Context) error {
	buckets := []string{"raw-input", "processed"}
	for _, b := range buckets {
		exists, err := s.client.BucketExists(ctx, b)
		if err != nil {
			return err
		}
		if !exists {
			if err := s.client.MakeBucket(ctx, b, minio.MakeBucketOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}
