# secrets-manager-api

This is a simple web application written in Go that provides a simple management tool for static secrets and TLS certificates using as a storage AWS Secrets Manager and AWS S3 for static files.
The application is hosted as a Lambda function behind an API Gateway.

The main dependencies are:
- aws-sdk-go-v2 - AWS SDK for Go v2 for interacting with AWS services
- chi - lightweight router for building Go HTTP services
- htmx - moder AJAX library for web applications
- alpine.js - lightweight JavaScript framework
- a-h templ - template engine for Go
- templui - UI components for a-h templ
- tailwindcss - utility-first CSS framework
