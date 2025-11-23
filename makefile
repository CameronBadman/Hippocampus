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


PYTHON_LAMBDAS := agent-curate safety-agent
LAMBDA_SRC_DIR := src/lambda
LAMBDA_BUILD_DIR := src/lambda/lambda-packages
TERRAFORM_LAMBDA_DIR := terraform/lambda-packages


build-python-lambdas:
	@echo "📦 Building Python Lambda packages..."
	@mkdir -p $(TERRAFORM_LAMBDA_DIR)
	@for lambda in $(PYTHON_LAMBDAS); do \
		echo "→ Packaging $$lambda ..."; \
		PACKAGE_DIR="$(LAMBDA_BUILD_DIR)/$$lambda"; \
		rm -rf $$PACKAGE_DIR; \
		mkdir -p $$PACKAGE_DIR; \
		cp $(LAMBDA_SRC_DIR)/$${lambda//-/_}_lambda.py $$PACKAGE_DIR/handler.py; \
		echo "requests" > $$PACKAGE_DIR/requirements.txt; \
		pip install -q --upgrade -r $$PACKAGE_DIR/requirements.txt --target $$PACKAGE_DIR; \
		cd $$PACKAGE_DIR && zip -rq $(abspath $(TERRAFORM_LAMBDA_DIR))/$$lambda.zip . && cd - >/dev/null; \
		echo "✓ Built $(TERRAFORM_LAMBDA_DIR)/$$lambda.zip"; \
	done


clean-python-lambdas:
	@echo "🧹 Cleaning Python Lambda build artifacts..."
	rm -rf $(LAMBDA_BUILD_DIR) $(TERRAFORM_LAMBDA_DIR)/*.zip

# Include Python Lambda packaging in full deploy pipeline
deploy: build-lambda build-python-lambdas
	@echo "🚀 Deploying to AWS with Terraform..."
	cd terraform && terraform apply

