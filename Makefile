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

.PHONY: install-backend
install-backend:
	go mod download

.PHONY: test-unit-backend
test-unit-backend:
	go test ./...

.PHONY: build-backend
build-backend:
	go build $(BUILD_OPTS) -o plugin-backend -mod=readonly cmd/plugin-backend.go

# docker inspect --format '{{.Architecture}}' quay.io/redhat-cne/openshift-origin-release:rhel-9-golang-1.23-openshift-4.19
# base image above has a arm64 archiecture; need to build executable /plugin-backend as amd64 to be match final container archiecture 
.PHONY: build-backend-dev
build-backend-dev:
	GOARCH=amd64 GOOS=linux go build -mod=readonly -o plugin-backend -mod=readonly cmd/plugin-backend.go

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
