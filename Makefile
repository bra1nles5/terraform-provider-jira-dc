PLUGIN_NAME=jira-dc
PLUGIN_VERSION=0.1.0
PLUGIN_NAMESPACE=bra1nles5
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
#PLUGIN_PATH=~/.terraform.d/plugins/$(PLUGIN_NAMESPACE)/$(PLUGIN_NAME)/$(PLUGIN_VERSION)/$(GOOS)_$(GOARCH)
PLUGIN_PATH=$(HOME)/.terraform.d/plugins/registry.terraform.io/$(PLUGIN_NAMESPACE)/$(PLUGIN_NAME)/$(PLUGIN_VERSION)/$(GOOS)_$(GOARCH)
BINARY_NAME=terraform-provider-$(PLUGIN_NAME)_v$(PLUGIN_VERSION)

## Developer-local targets, if present (not part of the repository)
-include local.mk

.PHONY: all build install clean test unit fmt lint

all: build install
run: clean build install test
rebuild: build install

## Build the provider binary
build:
	@echo "🧱 Building provider binary..."
	go mod tidy
	go build -ldflags "-X main.version=$(PLUGIN_VERSION)" -o $(BINARY_NAME)

## Install the binary into the local Terraform plugin directory
install: build
	@echo "📦 Installing provider to: $(PLUGIN_PATH)"
	mkdir -p $(PLUGIN_PATH)
	cp $(BINARY_NAME) $(PLUGIN_PATH)/
	@echo "✅ Installed: $(PLUGIN_PATH)/$(BINARY_NAME)"
	@echo "🚀 Initializing Terraform..."
	terraform init -reconfigure

## Code checks
fmt:
	@echo "🎨 Formatting Go code..."
	go fmt ./...

lint:
	@echo "🔍 Running go vet..."
	go vet ./...

## Unit tests
unit:
	@echo "🧪 Running unit tests..."
	go test ./...

## Local testing through Terraform
test: install
	@echo "⚙️  Running terraform plan..."
	TF_LOG=DEBUG TF_LOG_PROVIDER=TRACE terraform plan

## Clean up
clean:
	@echo "🧹 Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -rf $(PLUGIN_PATH)
	rm -rf .terraform .terraform.lock.hcl terraform.tfstate*