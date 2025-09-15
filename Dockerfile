# Frontend build stage
FROM oven/bun:1.1.42-alpine AS build-frontend
WORKDIR /app/frontend

# Copy package files first for better caching
COPY frontend/package.json frontend/bun.lockb* ./
RUN bun install --frozen-lockfile

# Copy source files and build
COPY frontend/ ./
RUN bun run build

# Go build stage
FROM golang:1.24-alpine AS build-go
ENV CGO_ENABLED=0
ARG BUILD_VERSION

# Install git for go mod operations
RUN apk add --no-cache git

WORKDIR /app

# Set up Go module cache directory
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-mod

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/root/.cache/go-mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy only Go source code and necessary files
COPY main.go ./
COPY pkg/ ./pkg/
COPY graphql/ ./graphql/

# Copy frontend embed.go file
COPY frontend/embed.go ./frontend/

# Copy frontend build output
COPY --from=build-frontend /app/frontend/dist /app/frontend/dist

# Build the application with cache mounts and embed version
RUN --mount=type=cache,target=/root/.cache/go-mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-w -s -X github.com/secmon-lab/lycaon/pkg/domain/types.Version=${BUILD_VERSION}" -o lycaon

# Final stage
FROM gcr.io/distroless/base:nonroot
USER nonroot
COPY --from=build-go /app/lycaon /lycaon

ENTRYPOINT ["/lycaon"]
