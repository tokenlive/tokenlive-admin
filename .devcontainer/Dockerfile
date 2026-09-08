FROM golang:1.27.1-bookworm AS go-toolchain

FROM mcr.microsoft.com/devcontainers/go:1-bookworm

COPY --from=go-toolchain /usr/local/go /usr/local/go1.27.1
ENV PATH="/usr/local/go1.27.1/bin:${PATH}"

ARG APP=tokenlive-admin

# Set CGO_CFLAGS to enable large file support
ENV CGO_CFLAGS "-D_LARGEFILE64_SOURCE"

RUN go install github.com/google/wire/cmd/wire@latest \
    && go install github.com/swaggo/swag/cmd/swag@latest \
    && go install github.com/gin-admin/gin-admin-cli/v10@latest \
    && chown -R vscode /go
