# syntax=docker/dockerfile:1
# Build context must be the examples/project directory (root of this example).
FROM golang:1.25-alpine AS builder
WORKDIR /build/project/src/service
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY src/service/go.mod src/service/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

COPY src/service/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	go build -ldflags='-extldflags=-static' -o /out/service ./cmd/service

FROM scratch
COPY --from=builder /out/service /service
ENTRYPOINT ["/service"]
