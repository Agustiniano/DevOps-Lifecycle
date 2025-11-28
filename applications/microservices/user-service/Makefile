GO_VERSION := 1.21
BINARY_NAME := api
DOCKER_REGISTRY := harbor.acme.com
IMAGE_NAME := platform/user-service
VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Install development dependencies
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install gotest.tools/gotestsum@latest

.PHONY: deps
deps: ## Download dependencies
	go mod download
	go mod verify

.PHONY: build
build: ## Build the binary
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s -X main.version=$(VERSION) -X main.commit=$(COMMIT)" \
		-trimpath \
		-o bin/$(BINARY_NAME) \
		./cmd/api

.PHONY: run
run: ## Run locally
	go run ./cmd/api

.PHONY: test
test: ## Run unit tests
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -v -tags=integration ./tests/integration/...

.PHONY: coverage
coverage: test ## Generate coverage report
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | grep total | awk '{print "Coverage: " $$3}'

.PHONY: lint
lint: ## Run linter
	golangci-lint run --timeout=5m ./...

.PHONY: security
security: ## Run security scan
	gosec -fmt json -out gosec-results.json ./...

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		-t $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest \
		.

.PHONY: docker-push
docker-push: ## Push Docker image
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest

.PHONY: docker-scan
docker-scan: ## Scan Docker image for vulnerabilities
	trivy image $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)

.PHONY: deploy-dev
deploy-dev: ## Deploy to dev environment
	kubectl apply -k ../../gitops/environments/overlays/dev/user-service/

.PHONY: deploy-staging
deploy-staging: ## Deploy to staging environment
	kubectl apply -k ../../gitops/environments/overlays/staging/user-service/

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f gosec-results.json

.DEFAULT_GOAL := help
