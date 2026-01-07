package acm

import (
	"context"
	"fmt"
	"strings"

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

func (s *Service) FindCertificateARNByDomain(ctx context.Context, domain string) (string, error) {
	s.log.Info("Searching for certificate with domain: %s", domain)
	out, err := s.client.ListCertificates(ctx, &acm.ListCertificatesInput{})
	if err != nil {
		return "", err
	}
	s.log.Info("Listed %d certificates", len(out.CertificateSummaryList))

	for _, certSummary := range out.CertificateSummaryList {
		if strings.Contains(*certSummary.DomainName, domain) {
			return *certSummary.CertificateArn, nil
		}
	}

	return "", fmt.Errorf("certificate not found for domain: %s", domain)
}
