# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.22

FROM alpine:${ALPINE_VERSION} AS runtime-base
RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S nylas \
	&& adduser -S -D -h /home/nylas -G nylas nylas \
	&& mkdir -p /home/nylas/.cache /home/nylas/.config \
	&& chown -R nylas:nylas /home/nylas

ENV HOME=/home/nylas \
	XDG_CACHE_HOME=/home/nylas/.cache \
	XDG_CONFIG_HOME=/home/nylas/.config \
	NYLAS_DISABLE_KEYRING=true

USER nylas
WORKDIR /home/nylas

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS source-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE

RUN --mount=type=cache,target=/root/.cache/go-build \
	build_date="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}" \
	&& CGO_ENABLED=0 GOOS="${TARGETOS:-linux}" GOARCH="${TARGETARCH:-$(go env GOARCH)}" go build -trimpath \
		-ldflags "-s -w -X github.com/nylas/cli/internal/version.Version=${VERSION} -X github.com/nylas/cli/internal/version.Commit=${COMMIT} -X github.com/nylas/cli/internal/version.BuildDate=${build_date}" \
		-o /out/nylas ./cmd/nylas

FROM runtime-base AS goreleaser
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/nylas /usr/local/bin/nylas
ENTRYPOINT ["/usr/local/bin/nylas"]
CMD ["--help"]

FROM runtime-base AS source
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="Nylas CLI" \
	org.opencontainers.image.description="CLI for Nylas API - manage email, calendar, and contacts" \
	org.opencontainers.image.source="https://github.com/nylas/cli" \
	org.opencontainers.image.version="${VERSION}" \
	org.opencontainers.image.revision="${COMMIT}" \
	org.opencontainers.image.created="${BUILD_DATE}"

COPY --from=source-builder /out/nylas /usr/local/bin/nylas
ENTRYPOINT ["/usr/local/bin/nylas"]
CMD ["--help"]
