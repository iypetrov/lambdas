package clusters

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
)

type Service struct {
	client *eks.Client
	log    logger.Logger
}

func NewService(ctx context.Context, cfg config.Config, log logger.Logger) *Service {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %v", err)
		return nil
	}

	return &Service{
		client: eks.NewFromConfig(awsCfg),
		log:    log,
	}
}

func (s *Service) GetAllClusters(ctx context.Context) ([]string, error) {
	clusters, err := s.client.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return []string{}, err
	}
	return clusters.Clusters, nil
}

