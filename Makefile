.PHONY: build
build: go build -o prometheus-deepflow-adapter ./cmd/

.PHONY: mod
vendor: go mod tidy