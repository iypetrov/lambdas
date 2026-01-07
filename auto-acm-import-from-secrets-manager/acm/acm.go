package acm

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/config"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/logger"
)

type Service struct {
	client *acm.Client
	log    logger.Logger
}

func NewService(ctx context.Context, cfg config.Config, log logger.Logger) *Service {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %v", err)
		return nil
	}

	return &Service{
		client: acm.NewFromConfig(awsCfg),
		log:    log,
	}
}

func (s *Service) ImportCert(ctx context.Context, cert, key string) (string, error) {
	result, err := s.client.ImportCertificate(ctx, &acm.ImportCertificateInput{
		Certificate: []byte(cert),
		PrivateKey:  []byte(key),
	})
	if err != nil {
		return "", err
	}
	return aws.StringValue(result.CertificateArn), nil
}
