FROM public.ecr.aws/docker/library/golang:1.22 AS build-stage
WORKDIR /app
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/main .

FROM scratch AS run-stage
COPY --from=build-stage /app/bin/main /bin/main
CMD ["/bin/main"]
