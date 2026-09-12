# Stage 1: Build frontend
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-builder

ARG VERSION=dev
ARG RELEASE_TAG=${VERSION}
ENV VITE_APP_VERSION=${RELEASE_TAG}

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN test "${RELEASE_TAG}" != "latest" && npm run build:prod

# Stage 2: Build backend
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS backend-builder

ARG APP=tokenlive-admin
ARG VERSION=dev
ARG RELEASE_TAG=${VERSION}
ARG BUILD_KIND=dev
ARG GOPROXY="https://proxy.golang.org,direct"

ENV GOPROXY=${GOPROXY}

ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/${APP}
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Copy frontend build output
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Build the application
RUN test "${RELEASE_TAG}" != "latest" && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-w -s -X main.VERSION=${RELEASE_TAG} -X main.BUILD_KIND=${BUILD_KIND}" -o ./${APP} .

# Stage 3: Production image
FROM alpine
ARG APP=tokenlive-admin

# Install ca-certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=backend-builder /go/src/${APP}/${APP} /usr/bin/${APP}

# Copy frontend static files
COPY --from=frontend-builder /app/frontend/dist /app/dist

# Copy configuration files
COPY configs /app/configs

EXPOSE 8040

ENTRYPOINT ["/usr/bin/tokenlive-admin", "start", "-d", "/app/configs", "-c", "prod", "-s", "/app/dist"]
