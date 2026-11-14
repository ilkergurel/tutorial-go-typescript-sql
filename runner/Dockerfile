# runner/Dockerfile builds the sandbox image: Go + Node toolchains that execute
# lesson code. This is the security boundary (ADR-0003): run as non-root with a
# read-only rootfs, no internet, and Compose-enforced cpu/mem/pids limits.
ARG GO_VERSION=1.27.4
ARG NODE_VERSION=22

FROM golang:${GO_VERSION} AS gobuild
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 go build -trimpath -o /out/runner ./cmd/runner

FROM node:${NODE_VERSION} AS nodebase

FROM nodebase
ARG GO_VERSION=1.27.4
# Go toolchain for `go build` of lesson code; NODE_ENV=development keeps npm
# installs of lesson dependencies fast.
COPY --from=gobuild /out/runner /usr/local/bin/runner
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
ENV GOTOOLCHAIN=local
USER node
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/runner"]
