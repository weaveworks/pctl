##@ Test

test: lint unit integration docs ## Lint, run all tests and update the docs

unit: ## Run the unit tests
	ginkgo -r ./pkg

integration: build test-env ## Run the integration tests
	ginkgo -r ./tests/...

test-env: ## Create an environment for tests
	cd dependencies/profiles && make docker-build-local kind-up docker-push-local
	flux install --components="source-controller,helm-controller,kustomize-controller"

##@ Build

lint: ## Run the linter
	golangci-lint run --exclude-use-default=false --timeout=5m0s

build: ## Build the pctl binary to ./pctl
	go build -o pctl ./cmd/pctl

local-env: ## Create local environment
	cd dependencies/profiles && make local-env
	kubectl apply -f dependencies/profiles/examples/profile-catalog-source.yaml

submodule: ## Update git submodules
	git submodule init
	git submodule update

##@ Docs

docs: mdtoc ## Update the Readme
	mdtoc -inplace README.md

mdtoc: ## Download mdtoc binary if necessary
	GO111MODULE=off go get sigs.k8s.io/mdtoc || true

##@ Utility

.PHONY: help
help:  ## Display this help. Thanks to https://www.thapaliya.com/en/writings/well-documented-makefiles/
ifeq ($(OS),Windows_NT)
		@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make <target>\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-30s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
else
		@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
endif
