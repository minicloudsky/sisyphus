.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest

.PHONY: api
api:
	protoc --proto_path=./api \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./api \
		--go-http_out=paths=source_relative:./api \
		--go-grpc_out=paths=source_relative:./api \
		./api/proto/v1/*.proto

.PHONY: proto
proto: api

.PHONY: build
build:
	go build -o bin/sisyphus ./cmd/sisyphus

.PHONY: wire
wire:
	cd cmd/sisyphus && wire

.PHONY: run
run:
	go run ./cmd/sisyphus -conf configs/config.yaml

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin/
	rm -f cmd/sisyphus/wire_gen.go

# ── Docker ────────────────────────────────────────────────────────
REGISTRY ?= docker.io/your-org
APP_TAG  ?= latest

# Build the backend image (single Dockerfile, all stages in one pass)
.PHONY: docker-build
docker-build:
	docker build \
		-t sisyphus-server:$(APP_TAG) \
		-t $(REGISTRY)/sisyphus-server:$(APP_TAG) \
		.

# Push to registry
.PHONY: docker-push
docker-push:
	docker push $(REGISTRY)/sisyphus-server:$(APP_TAG)

# Build + push in one shot
.PHONY: docker-release
docker-release: docker-build docker-push
