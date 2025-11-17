package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
    "github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
)


func init() {
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log := logger.Get(ctx)
	log.Info("Handling request: %v", event.Path)

	var chiLambda *chiadapter.ChiLambda
	r := chi.NewRouter()
	
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from /test"))
	})
	
	r.Get("/secrets/{name}", func(w http.ResponseWriter, r *http.Request) {
		secretName := chi.URLParam(r, "name")
		w.Write([]byte("Requested secret: " + secretName))
	})

	chiLambda = chiadapter.New(r)
	return chiLambda.ProxyWithContext(ctx, event)
}

func main() {
	ctx := context.Background()
	cfg := config.New()
	log := logger.New(cfg)
	ctx = log.Inject(ctx)
	ctx = config.Inject(ctx, *cfg)

	lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
}
