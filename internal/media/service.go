package media

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/tasks"
	"github.com/hibiken/asynq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MediaService struct {
	repo        MediaRepository
	minioClient *minio.Client
	queue       *asynq.Client
	config      *config.AllConfiguration
	bucketName  string
}

func NewMediaService(repo MediaRepository, queue *asynq.Client, cfg *config.AllConfiguration) (*MediaService, error) {
	// Initialize MinIO client object
	// Ensure endpoint doesn't have http schema for MinIO client New
	endpoint := strings.ReplaceAll(cfg.AWS.Endpoint, "http://", "")
	endpoint = strings.ReplaceAll(endpoint, "https://", "")

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AWS.AccessKeyID, cfg.AWS.SecretAccessKey, ""),
		Secure: false, // Set to true for HTTPS
	})
	if err != nil {
		return nil, err
	}

	return &MediaService{
		repo:        repo,
		minioClient: minioClient,
		queue:       queue,
		config:      cfg,
		bucketName:  cfg.AWS.BucketName,
	}, nil
}

func (s *MediaService) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*Media, error) {
	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	uniqueName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload to MinIO
	info, err := s.minioClient.PutObject(ctx, s.bucketName, uniqueName, file, header.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}

	// Construct Public URL (For local minio, we might need a presigned URL or just a proxy URL)
	// For this setup, we assume public bucket.
	// We need to construct URL based on external access.
	// For local development, minio is at port 9000, but from browser it might be different.
	// Let's use the configurated endpoint.
	// Generate Public URL
	publicEndpoint := s.config.AWS.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = s.config.AWS.Endpoint
	}
	// Ensure scheme if missing (basic check)
	if !strings.HasPrefix(publicEndpoint, "http") {
		publicEndpoint = "http://" + publicEndpoint
	}

	url := fmt.Sprintf("%s/%s/%s", publicEndpoint, s.bucketName, info.Key)

	media := &Media{
		FileName:     info.Key,
		OriginalName: header.Filename,
		MimeType:     header.Header.Get("Content-Type"),
		SizeBytes:    info.Size,
		URL:          url,
		BucketName:   s.bucketName,
		Region:       s.config.AWS.Region,
	}

	createdMedia, err := s.repo.Create(ctx, media)
	if err != nil {
		return nil, err
	}

	// Enqueue background processing task
	task, err := tasks.NewProcessImageTask(createdMedia.ID.String())
	if err == nil {
		// Log error if enqueue fails, but don't fail the request (or handle as needed)
		if _, err := s.queue.Enqueue(task); err != nil {
			// In a real app, use logger here
			fmt.Printf("Failed to enqueue image processing task: %v\n", err)
		}
	}

	return createdMedia, nil
}
