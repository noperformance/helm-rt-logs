HELM_PLUGIN_NAME := rt-logs
LDFLAGS := "-X main.version=${VERSION}"

.PHONY: build
build:
	export CGO_ENABLED=0 && \
	go build -o bin/${HELM_PLUGIN_NAME} -ldflags $(LDFLAGS) .

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: vet

.PHONY: test
test:
	go test -v ./...

.PHONY: tag
tag:
	@scripts/tag.sh
