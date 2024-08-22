# Build variables
BINARY_NAME=webhook
BUILD_DIR=bin
DOCKER_IMAGE=admission-webhook
DOCKER_TAG=latest

# Go variables
GO=go
GOFMT=gofmt
GOOS=linux
GOARCH=amd64

# Directories
CMD_DIR=cmd/webhook
PKG_DIR=pkg

.PHONY: all build clean test docker-build docker-push deploy undeploy certs help

all: build

## Build the webhook binary
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

## Run tests
test:
	$(GO) test -v ./$(PKG_DIR)/...

## Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./$(PKG_DIR)/...
	$(GO) tool cover -html=coverage.out -o coverage.html

## Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## Push Docker image (uncomment and customize as needed)
# docker-push:
# 	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

## Generate certificates
certs:
	cd certs && ./generate.sh

## Deploy to Kubernetes
deploy: docker-build certs
	kubectl apply -f deploy/namespace.yaml
	kubectl apply -f deploy/rbac.yaml
	kubectl apply -f deploy/webhook-certs-secret.yaml
	kubectl apply -f deploy/deployment.yaml
	kubectl apply -f deploy/service.yaml
	kubectl apply -f deploy/validating-webhook-config.yaml
	kubectl apply -f deploy/mutating-webhook-config.yaml

## Undeploy from Kubernetes
undeploy:
	kubectl delete -f deploy/validating-webhook-config.yaml
	kubectl delete -f deploy/mutating-webhook-config.yaml
	kubectl delete -f deploy/service.yaml
	kubectl delete -f deploy/deployment.yaml
	kubectl delete -f deploy/rbac.yaml
	kubectl delete -f deploy/namespace.yaml
	kubectl delete secret webhook-certs -n webhook-system || true

## Run locally with development settings
run-dev:
	$(GO) run $(CMD_DIR)/main.go

## Format code
fmt:
	$(GOFMT) -s -w .

## Vet code
vet:
	$(GO) vet ./...

## Run all checks
check: fmt vet test

## Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the webhook binary"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  docker-build   - Build Docker image"
	@echo "  certs          - Generate TLS certificates"
	@echo "  deploy         - Deploy to Kubernetes"
	@echo "  undeploy       - Undeploy from Kubernetes"
	@echo "  run-dev        - Run locally for development"
	@echo "  fmt            - Format code"
	@echo "  vet            - Vet code"
	@echo "  check          - Run all checks"
	@echo "  help           - Show this help"
# Build
