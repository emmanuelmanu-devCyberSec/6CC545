# Run docker compose from the submission root, which contains source/.
# Pin a staff-approved supported image and record its digest for release.
ARG GO_IMAGE=golang:1.27.3-bookworm
FROM ${GO_IMAGE} AS build
WORKDIR /src
COPY source/go.mod source/go.sum ./
RUN go mod download
COPY source/ ./
ARG APP_PACKAGE=./cmd/deploycheck
RUN CGO_ENABLED=0 go build -trimpath -o /out/node ${APP_PACKAGE}
RUN CGO_ENABLED=0 go build -trimpath -o /out/probe ./cmd/probe

# The supplied Go build image contains a C toolchain for the race-test target.
# Smoke/regression target: deliberately excludes assessed feature tests.
FROM build AS test
RUN CGO_ENABLED=1 go test -race ./... -count=1 -timeout 60s

# Feature target: EXPECTED TO FAIL until student TODOs and wiring are complete.
FROM build AS acceptance
RUN CGO_ENABLED=1 go test -race -tags acceptance ./... -count=1 -timeout 120s

FROM debian:bookworm-slim AS runtime
RUN mkdir -p /app /data && chown -R 65532:65532 /app /data
COPY --from=build /out/node /app/node
COPY --from=build /out/probe /app/probe
USER 65532:65532
WORKDIR /app
EXPOSE 8080 9090 8443 9443 9444
ENTRYPOINT ["/app/node"]
