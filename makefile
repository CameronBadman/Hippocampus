.PHONY: build-cli build-lambda clean test deploy all

build-cli:
	@echo "Building CLI..."
	@mkdir -p bin
	CGO_ENABLED=0 go build -o bin/hippocampus src/cmd/cli/main.go
	@echo "✓ CLI built: bin/hippocampus"

build-lambda:
	@echo "Building Lambda function..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-tags lambda.norpc \
		-o terraform/bootstrap \
		src/lambda/main.go
	@echo "✓ Lambda built: terraform/bootstrap"

clean:
	rm -rf bin/ terraform/bootstrap terraform/lambda.zip terraform/.terraform*

test:
	go test ./src/...

deploy: build-lambda
	@echo "Deploying to AWS..."
	cd terraform && terraform apply

all: build-cli build-lambda

.DEFAULT_GOAL := build-cli
