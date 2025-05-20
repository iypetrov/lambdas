go-bootstrap-linux:
	@cp -r go-hello "$(LAMBDA)"
	@find "$(LAMBDA)" -type f -exec sed -i "s/go-hello/$(LAMBDA)/g" {} +

go-bootstrap-mac:
	@cp -r go-hello "$(LAMBDA)"
	@find "$(LAMBDA)" -type f -exec sed -i '' "s/go-hello/$(LAMBDA)/g" {} +

python-bootstrap-linux:
	@cp -r python-hello "$(LAMBDA)"
	@find "$(LAMBDA)" -type f -exec sed -i "s/python-hello/$(LAMBDA)/g" {} +

python-bootstrap-mac:
	@cp -r python-hello "$(LAMBDA)"
	@find "$(LAMBDA)" -type f -exec sed -i '' "s/python-hello/$(LAMBDA)/g" {} +
