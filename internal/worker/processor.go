package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/media"
	"github.com/hfleury/horsemarketplacebk/internal/tasks"
	"github.com/hibiken/asynq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Processor struct {
	repo        media.MediaRepository
	minioClient *minio.Client
	config      *config.AllConfiguration
	logger      config.Logging
}

func NewProcessor(repo media.MediaRepository, cfg *config.AllConfiguration, logger config.Logging) (*Processor, error) {
	endpoint := strings.ReplaceAll(cfg.AWS.Endpoint, "http://", "")
	endpoint = strings.ReplaceAll(endpoint, "https://", "")

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AWS.AccessKeyID, cfg.AWS.SecretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	return &Processor{
		repo:        repo,
		minioClient: minioClient,
		config:      cfg,
		logger:      logger,
	}, nil
}

func (p *Processor) HandleProcessImageTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.ProcessImagePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	p.logger.Log(nil, config.InfoLevel, "Processing image task", map[string]any{"media_id": payload.MediaID})

	mediaID, err := uuid.Parse(payload.MediaID)
	if err != nil {
		return fmt.Errorf("start processing: invalid media id: %w", err)
	}

	m, err := p.repo.FindByID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("start processing: find media: %w", err)
	}
	if m == nil {
		return fmt.Errorf("media not found: %s", payload.MediaID)
	}

	// Download Original
	obj, err := p.minioClient.GetObject(ctx, m.BucketName, m.FileName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download object: %w", err)
	}
	defer obj.Close()

	img, _, err := image.Decode(obj)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	// Resize logic
	variants := make(map[string]string)

	// Create Thumbnail
	thumb := imaging.Thumbnail(img, 150, 150, imaging.Lanczos)
	thumbUrl, err := p.uploadVariant(ctx, thumb, m.FileName, "thumb")
	if err != nil {
		return err
	}
	variants["thumbnail"] = thumbUrl

	// Create Medium
	medium := imaging.Resize(img, 800, 0, imaging.Lanczos)
	mediumUrl, err := p.uploadVariant(ctx, medium, m.FileName, "medium")
	if err != nil {
		return err
	}
	variants["medium"] = mediumUrl

	// Update DB
	variantsJSON, _ := json.Marshal(variants)
	if err := p.repo.UpdateVariants(ctx, m.ID, variantsJSON); err != nil {
		return fmt.Errorf("update variants: %w", err)
	}

	p.logger.Log(nil, config.InfoLevel, "Image processing completed", map[string]any{"media_id": payload.MediaID})
	return nil
}

func (p *Processor) uploadVariant(ctx context.Context, img image.Image, originalName, suffix string) (string, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return "", fmt.Errorf("encode jpeg: %w", err)
	}

	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)
	newName := fmt.Sprintf("%s-%s.jpg", nameWithoutExt, suffix)

	// Upload
	_, err := p.minioClient.PutObject(ctx, p.config.AWS.BucketName, newName, buf, int64(buf.Len()), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return "", fmt.Errorf("upload variant: %w", err)
	}

	// Generate Public URL
	publicEndpoint := p.config.AWS.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = p.config.AWS.Endpoint
	}
	if !strings.HasPrefix(publicEndpoint, "http") {
		publicEndpoint = "http://" + publicEndpoint
	}

	return fmt.Sprintf("%s/%s/%s", publicEndpoint, p.config.AWS.BucketName, newName), nil
}
