package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSClient wraps Google Cloud Storage operations
type GCSClient struct {
	client     *storage.Client
	bucketName string
}

// NewGCSClient creates a new GCS client
func NewGCSClient() (*GCSClient, error) {
	ctx := context.Background()

	bucketName := os.Getenv("GCS_BUCKET")
	if bucketName == "" {
		bucketName = "sago-pitch-decks"
	}

	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		credsFile = "service-account.json"
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSClient{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadFile uploads a file to GCS and returns the GCS path
func (g *GCSClient) UploadFile(ctx context.Context, objectName string, data []byte) (string, error) {
	bucket := g.client.Bucket(g.bucketName)
	obj := bucket.Object(objectName)

	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/pdf"

	if _, err := writer.Write(data); err != nil {
		return "", fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	gcsPath := fmt.Sprintf("gs://%s/%s", g.bucketName, objectName)
	return gcsPath, nil
}

// DownloadFile downloads a file from GCS
func (g *GCSClient) DownloadFile(ctx context.Context, objectName string) ([]byte, error) {
	bucket := g.client.Bucket(g.bucketName)
	obj := bucket.Object(objectName)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read from GCS: %w", err)
	}

	return data, nil
}

// GetSignedURL generates a signed URL for temporary access
func (g *GCSClient) GetSignedURL(objectName string, expiration time.Duration) (string, error) {
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}

	url, err := g.client.Bucket(g.bucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// DeleteFile deletes a file from GCS
func (g *GCSClient) DeleteFile(ctx context.Context, objectName string) error {
	bucket := g.client.Bucket(g.bucketName)
	obj := bucket.Object(objectName)
	return obj.Delete(ctx)
}

// Close closes the GCS client
func (g *GCSClient) Close() error {
	return g.client.Close()
}
