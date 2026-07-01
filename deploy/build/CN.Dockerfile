# Stage 1: Build frontend
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-builder
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm config set registry https://registry.npmmirror.com && npm ci
COPY frontend/ .
RUN npm run build:prod

# Stage 2: Build backend
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend-builder
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

ARG APP=tokenlive-admin
ARG VERSION=v1.0.0
ARG RELEASE_TAG=${VERSION}
ARG GOPROXY="https://goproxy.cn,direct"

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
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-w -s -X main.VERSION=${RELEASE_TAG}" -o ./${APP} .

# Stage 3: Production image
FROM alpine
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
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
