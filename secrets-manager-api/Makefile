run-local-linux-x64:
	@templ generate --watch --proxy="http://localhost:8080" --open-browser=false & \
	air -c .air.toml & \
	./bin/tailwindcss-linux-x64 -i ./static/css/input.css -o ./static/css/output.css --watch

run-local-linux-arm64:
	@templ generate --watch --proxy="http://localhost:8080" --open-browser=false & \
	air -c .air.toml & \
	./bin/tailwindcss-linux-arm64 -i ./static/css/input.css -o ./static/css/output.css --watch

run-local-mac-arm64:
	@templ generate --watch --proxy="http://localhost:8080" --open-browser=false & \
	air -c .air.toml & \
	./bin/tailwindcss-macos-arm64 -i ./static/css/input.css -o ./static/css/output.css --watch

fmt:
	@go fmt $(shell go list ./... | grep -v /vendor/)
	@find . -path ./vendor -prune -o -name '*.go' -exec goimports -l -w {} +

update-deps:
	@echo "Updating dependencies..."
	@curl -sL https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.8/tailwindcss-linux-x64 -o bin/tailwindcss-linux-x64
	@chmod +x bin/tailwindcss-linux-x64
	@echo "Updated tailwindcss-linux-x64"
	@curl -sL https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.8/tailwindcss-linux-arm64 -o bin/tailwindcss-linux-arm64
	@chmod +x bin/tailwindcss-linux-arm64
	@echo "Updated tailwindcss-linux-arm64"
	@curl -sL https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.8/tailwindcss-macos-arm64 -o bin/tailwindcss-macos-arm64
	@chmod +x bin/tailwindcss-macos-arm64
	@echo "Updated tailwindcss-macos-arm64"
	@curl -sL https://unpkg.com/htmx.org@2.0.3/dist/htmx.min.js -o static/js/htmx.min.js
	@echo "Updated htmx.min.js"
	@curl -sL https://cdn.jsdelivr.net/npm/alpinejs@3.14.3/dist/cdn.min.js -o static/js/alpine.min.js
	@echo "Updated alpine.min.js"
	@templui -f init
	@echo "Updated templui"
	@go get -u
	@echo "Updated Go dependencies"
	@go mod tidy
	@echo "Go modules tidied"
	@echo "Dependencies updated successfully."

vuln-check:
	@govulncheck -show verbose ./...
