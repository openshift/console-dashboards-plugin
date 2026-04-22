VERSION     ?= latest
PLATFORMS   ?= linux/arm64,linux/amd64
ORG         ?= openshift-observability-ui
IMAGE		?= quay.io/${ORG}/console-dashboards-plugin:${VERSION}

# Tools
GOLANGCI_LINT = $(shell pwd)/_output/tools/bin/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.61.0

.PHONY: install-frontend
install-frontend:
	cd web && npm install

.PHONY: install-frontend-ci
install-frontend-ci:
	cd web && npm ci --ignore-scripts

.PHONY: install-frontend-ci-clean
install-frontend-ci-clean: install-frontend-ci
	cd web && npm cache clean --force

.PHONY: build-frontend
build-frontend:
	cd web && npm run build

.PHONY: start-frontend
start-frontend:
	cd web && npm run start

.PHONY: start-console
start-console:
	./scripts/start-console.sh

.PHONY: lint-frontend
lint-frontend:
	cd web && npm run lint

# Download and install golangci-lint if not already installed
.PHONY: golangci-lint
golangci-lint:
	@[ -f $(GOLANGCI_LINT) ] || { \
		set -e ;\
		mkdir -p $(shell dirname $(GOLANGCI_LINT)) ;\
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) $(GOLANGCI_LINT_VERSION) ;\
	}

.PHONY: lint-backend
lint-backend: golangci-lint
	go mod tidy
	$(GOLANGCI_LINT) -c $(shell pwd)/.golangci.yml run --verbose --fix

.PHONY: install-backend
install-backend:
	go mod download

.PHONY: test-unit-backend
test-unit-backend:
	go test ./...

.PHONY: build-backend
build-backend:
	go build $(BUILD_OPTS) -o plugin-backend -mod=readonly cmd/plugin-backend.go

.PHONY: start-backend
start-backend:
	go run ./cmd/plugin-backend.go

.PHONY: build-image
build-image:
	./scripts/build-image.sh

.PHONY: install
install: install-frontend install-backend

.PHONY: example
example:
	cd docs && oc apply -f prometheus-datasource-example.yaml && oc apply -f prometheus-dashboard-example.yaml

.PHONY: podman-cross-build
podman-cross-build:
	podman manifest rm ${IMAGE} || true
	podman manifest create ${IMAGE}
	podman build --platform=${PLATFORMS} --manifest ${IMAGE} -f Dockerfile.dev
	podman manifest push ${IMAGE}
