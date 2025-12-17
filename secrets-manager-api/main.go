package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/clusters"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/images"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
	"github.com/iypetrov/lambdas/secrets-manager-api/metadata"
	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
)

func configServer(ctx context.Context, cfg config.Config, log logger.Logger) *chi.Mux {
	secretService := secrets.NewService(ctx, cfg, log)
	clusterService := clusters.NewService(ctx, cfg, log)
	imagesService := images.NewService(ctx, cfg, log)
	metadataService := metadata.NewService(ctx, cfg, log)

	handler := RouterHandler{
		config:         cfg,
		log:            log,
		secretsService: secretService,
		clusterService: clusterService,
		imagesService:      imagesService,
		metadataService:   metadataService,
	}

	return NewRouter(&handler)
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log := logger.Get(ctx)
	cfg := config.Get(ctx)
	var chiLambda = chiadapter.New(configServer(ctx, cfg, log))
	return chiLambda.ProxyWithContext(ctx, event)
}

func main() {
	ctx := context.Background()
	cfg := config.New()
	log := logger.New(cfg)

	log.Info("Starting Secrets Manager API in %s environment", cfg.App.Env)
	if cfg.App.Env == config.Local {
		if err := http.ListenAndServe(":8080", configServer(ctx, cfg, log)); err != nil {
			log.Error("Failed to start server: %v", err)
		}
	} else {
		ctx = log.Inject(ctx)
		ctx = config.Inject(ctx, cfg)
		lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
	}
}
