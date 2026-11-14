# server/Dockerfile builds both Go programs: backend and mocksvc.
# Build once, two images via the SERVICE build arg; both are distroless-style
# minimal runtime images (no shell, no package manager) per ADR-0003.
ARG GO_VERSION=1.27.4

FROM golang:${GO_VERSION} AS build
ARG SERVICE=stacklab
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian13
ARG SERVICE=stacklab
COPY --from=build /out/app /app
# Migrations are embedded in the binary (go:embed); no extra copy or volume.
USER nonroot:nonroot
ENTRYPOINT ["/app"]
