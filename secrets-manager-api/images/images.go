package images

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
)

var (
	ErrBucketNotConfigured = errors.New("S3 bucket is not configured")
)

type Service struct {
	client *s3.Client
	bucket string
	log    logger.Logger
}

func NewService(ctx context.Context, cfg config.Config, log logger.Logger) *Service {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %v", err)
		return nil
	}

	return &Service{
		client: s3.NewFromConfig(awsCfg),
		bucket: cfg.S3.Bucket,
		log:    log,
	}
}

func (s *Service) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.bucket == "" {
		return nil, ErrBucketNotConfigured
	}

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, getObjectInput)
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}

func (s *Service) ServeStaticFile(w http.ResponseWriter, r *http.Request, filePath string) error {
	ctx := r.Context()

	if strings.HasPrefix(filePath, "/") {
		filePath = filePath[1:]
	}
	s3Key := path.Join("static", filePath)

	body, err := s.GetObject(ctx, s3Key)
	if err != nil {
		s.log.Error("Failed to get object from S3: %v", err)
		return err
	}
	defer body.Close()

	contentType := GetContentType(filePath)
	w.Header().Set("Content-Type", contentType)

	if _, err := io.Copy(w, body); err != nil {
		s.log.Error("Failed to copy S3 object to response: %v", err)
		return err
	}

	return nil
}



func GetContentType(filePath string) string {
	ext := strings.ToLower(path.Ext(filePath))
	
	switch ext {
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".woff", ".woff2":
		return "font/woff"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".html", ".htm":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".xml":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}

func IsImage(filePath string) bool {
	ext := strings.ToLower(path.Ext(filePath))
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".bmp"}
	
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

func IsValidImageExtension(ext string) bool {
	ext = strings.ToLower(ext)
	validExts := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".bmp"}
	
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}
