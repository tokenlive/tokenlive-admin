.PHONY: start build build-frontend build-all build-cross-all clean

NOW = $(shell date -u '+%Y%m%d%I%M%S')

RELEASE_VERSION = v1.0.0

APP 		= tokenlive-admin
SERVER_BIN  	= bin/${APP}
GIT_COUNT 		= $(shell git rev-list --all --count)
GIT_HASH        = $(shell git rev-parse --short HEAD)
RELEASE_TAG     = $(RELEASE_VERSION).$(GIT_COUNT).$(GIT_HASH)

CONFIG_DIR       = ./configs
CONFIG_FILES     = dev
STATIC_DIR       = ./frontend/dist
START_ARGS       = -d $(CONFIG_DIR) -c $(CONFIG_FILES) -s $(STATIC_DIR)

all: start

start:
	@go run -ldflags "-X main.VERSION=$(RELEASE_TAG)" main.go start $(START_ARGS)

build:
	@go build -ldflags "-w -s -X main.VERSION=$(RELEASE_TAG)" -o $(SERVER_BIN)

build-frontend:
	@echo "Building frontend..."
	@cd frontend && npm ci && npm run build:prod
	@echo "Frontend build completed."

build-all: build-frontend build
	@echo "Full build completed. Frontend: $(STATIC_DIR), Backend: $(SERVER_BIN)"

# --- Cross-compilation targets (CGO_ENABLED=0, pure Go) ---
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s -X main.VERSION=$(RELEASE_TAG)" -o $(SERVER_BIN)_linux_amd64

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-w -s -X main.VERSION=$(RELEASE_TAG)" -o $(SERVER_BIN)_linux_arm64

build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-w -s -X main.VERSION=$(RELEASE_TAG)" -o $(SERVER_BIN)_darwin_amd64

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-w -s -X main.VERSION=$(RELEASE_TAG)" -o $(SERVER_BIN)_darwin_arm64

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-w -s -X main.VERSION=$(RELEASE_TAG)" -o $(SERVER_BIN)_windows_amd64.exe

build-cross-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	@echo "All cross-platform binaries built."

# go install github.com/google/wire/cmd/wire@latest
wire:
	@wire gen ./internal/wirex

# go install github.com/swaggo/swag/cmd/swag@latest
swagger:
	@swag init --parseDependency --generalInfo ./main.go --output ./internal/swagger

# https://github.com/OpenAPITools/openapi-generator
openapi:
	docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate -i /local/internal/swagger/swagger.yaml -g openapi -o /local/internal/swagger/v3

clean:
	rm -rf data $(SERVER_BIN) $(SERVER_BIN)_* frontend/dist

serve: build-all
	./$(SERVER_BIN) start $(START_ARGS)

serve-d: build-all
	./$(SERVER_BIN) start $(START_ARGS) --daemon

stop:
	./$(SERVER_BIN) stop

gen-application-code:
	gin-admin-cli gen -d ./ -m Resource -c gen/prototype/application.yaml

docker-build:
	docker build -t $(APP):$(RELEASE_TAG) .
	docker tag $(APP):$(RELEASE_TAG) $(APP):latest

docker-push: docker-build
	docker push $(APP):$(RELEASE_TAG)
	docker push $(APP):latest