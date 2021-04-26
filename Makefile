##@ Test

test: lint unit-test integration-test ## Run all tests

unit-test: ## Run the unit tests
	ginkgo -r ./pkg

integration-test: build local-env ## Run the integration tests
	ginkgo -r ./tests/...

local-env: submodule
	cd dependencies/profiles && make local-env
	kubectl apply -f dependencies/profiles/examples/profile-catalog-source.yaml

submodule:
	git submodule init
	git submodule update

##@ Build

lint:
	golangci-lint run --exclude-use-default=false --timeout=5m0s ## Run the linter

build: ## Build the pctl binary to ./pctl
	go build -o pctl ./cmd/pctl

##@ Utility

.PHONY: help
help:  ## Display this help. Thanks to https://www.thapaliya.com/en/writings/well-documented-makefiles/
ifeq ($(OS),Windows_NT)
		@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make <target>\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-30s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
else
		@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
endif

