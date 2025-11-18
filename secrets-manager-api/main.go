package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"

	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
)

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log := logger.Get(ctx)
	cfg := config.Get(ctx)
	var chiLambda *chiadapter.ChiLambda

	handler := RouterHandler{
		config:        cfg,
		log:           log,
	}
	r := NewRouter(handler)

	chiLambda = chiadapter.New(r)
	return chiLambda.ProxyWithContext(ctx, event)
}

func main() {
	ctx := context.Background()
	cfg := config.New()
	log := logger.New(cfg)

	if cfg.App.Env == config.Local {
		handler := RouterHandler{
			config:        cfg,
			log:           log,
		}
		r := NewRouter(handler)
	 	http.ListenAndServe(":8080", r)
	} else {
		ctx = log.Inject(ctx)
		ctx = config.Inject(ctx, cfg)
		lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
	}
}
