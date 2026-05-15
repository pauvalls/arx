# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=0 go build -o /arx ./cmd/arx

# Runtime stage
FROM gcr.io/distroless/base-debian12

LABEL maintainer="Pau Valls"
LABEL source="https://github.com/pauvalls/arx"
LABEL license="MIT"

COPY --from=builder /arx /arx

ENTRYPOINT ["/arx"]
