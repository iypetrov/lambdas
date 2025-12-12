FROM golang:1.24.10 AS build-stage
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/main .

FROM public.ecr.aws/lambda/provided:al2.2025.11.18.06 AS run-stage
COPY --from=build-stage /app/bin/main /bin/main
ENTRYPOINT ["/bin/main"]
