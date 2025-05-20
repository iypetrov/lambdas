run-local:
	@docker build -f Dockerfile.local -t python-hello:test .
	@docker run -p 9000:8080 python-hello:test

invoke-local:
	curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" \
	-H "Content-Type: application/json" \
	--data-binary @payload.json
