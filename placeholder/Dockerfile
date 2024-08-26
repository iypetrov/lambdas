FROM golang:1.22 AS build-stage
WORKDIR /app
ADD . .
RUN go build -tags lambda.norpc -o bin/main main.go

FROM public.ecr.aws/lambda/provided:al2023 AS run-stage
COPY --from=build-stage /app/bin/main ./bin/main
ENTRYPOINT [ "./bin/main" ]
