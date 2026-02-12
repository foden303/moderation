GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: api
# generate api proto
api:
	protoc --proto_path=./api \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./api \
 	       --go-http_out=paths=source_relative:./api \
 	       --go-grpc_out=paths=source_relative:./api \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       --experimental_allow_proto3_optional \
	       $(API_PROTO_FILES)

.PHONY: nsfw-image-proto
# generate nsfw detector go client from deployments proto
nsfw-image-proto:
	mkdir -p api/nsfw_image/v1
	protoc --proto_path=./deployments/nsfw_image/proto \
	       --go_out=paths=source_relative:./api/nsfw_image/v1 \
	       --go-grpc_out=paths=source_relative:./api/nsfw_image/v1 \
	       nsfw_image.proto

.PHONY: nsfw-text-proto
# generate nsfw detector go client from deployments proto
nsfw-text-proto:
	mkdir -p api/nsfw_text/v1
	protoc --proto_path=./deployments/nsfw_text/proto \
	       --go_out=paths=source_relative:./api/nsfw_text/v1 \
	       --go-grpc_out=paths=source_relative:./api/nsfw_text/v1 \
	       nsfw_text.proto

.PHONY: nsfw-image-build
# build nsfw-image docker image
nsfw-image-build:
	docker compose -f docker-compose.dev.yml build nsfw-image

.PHONY: nsfw-text-build
# build nsfw-text docker image
nsfw-text-build:
	docker compose -f docker-compose.dev.yml build nsfw-text

.PHONY: nsfw-build-all
# build all nsfw docker images
nsfw-build-all: nsfw-image-build nsfw-text-build

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: run
# run server locally
run:
	go run ./cmd/storage -conf ./configs

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: dev-up
# start dev infrastructure (postgres, redis, vllm, nsfw-detector)
dev-up:
	docker compose -f docker-compose.dev.yml up -d

.PHONY: dev-down
# stop dev infrastructure
dev-down:
	docker compose -f docker-compose.dev.yml down

.PHONY: dev-logs
# show dev infrastructure logs
dev-logs:
	docker compose -f docker-compose.dev.yml logs -f

.PHONY: dev-ps
# show dev infrastructure status
dev-ps:
	docker compose -f docker-compose.dev.yml ps

.PHONY: dev-infra-lite
# start only postgres and redis (no GPU services)
dev-infra-lite:
	docker compose -f docker-compose.dev.yml up -d postgres redis

.PHONY: dev-reset
# reset dev infrastructure (remove volumes)
dev-reset:
	docker compose -f docker-compose.dev.yml down -v

.PHONY: all
# generate all
all:
	make api
	make config
	make generate

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)


.PHONY: migrate-up
# run migrations up
migrate-up:
	migrate -path internal/data/migrations -database "postgres://postgres:123456@localhost:5432/moderation?sslmode=disable" up

.PHONY: migrate-down
# run migrations down
migrate-down:
	migrate -path internal/data/migrations -database "postgres://postgres:123456@localhost:5432/moderation?sslmode=disable" down 1

.PHONY: redo-migrate
# reset database and run migrations up
redo-migrate:
	migrate -path internal/data/migrations -database "postgres://postgres:123456@localhost:5432/moderation?sslmode=disable" down -all
	migrate -path internal/data/migrations -database "postgres://postgres:123456@localhost:5432/moderation?sslmode=disable" up

.DEFAULT_GOAL := help
