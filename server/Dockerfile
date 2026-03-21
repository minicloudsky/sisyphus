# ═════════════════════════════════════════════════════════════════
# Sisyphus Backend — Application Dockerfile
#
# Builds FROM the pre-baked base image which already contains all
# OS packages and third-party tools (gh-ost, goInception, etc.).
# This keeps CI build time short — only Go compilation happens here.
#
# Stages
# ─────────────────────────────────────────────────────────────────
#   builder    Compile the Go binary (static, CGO_ENABLED=0)
#   runtime    Copy binary + config on top of the base image
#
# Build:
#   docker build \
#     --build-arg BASE_IMAGE=<dockerhub-user>/sisyphus-backend-base:1.0.0 \
#     -t <dockerhub-user>/sisyphus-backend:latest .
#
# Local build (without a pre-pushed base):
#   docker build -f Dockerfile.base -t sisyphus-backend-base:local .
#   docker build --build-arg BASE_IMAGE=sisyphus-backend-base:local \
#     -t sisyphus-backend:dev .
# ═════════════════════════════════════════════════════════════════

# ─────────────────────────────────────────────────────────────────
# Stage 1 — Compile the Go service binary
# ─────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /build

# Resolve modules before copying source — cached until go.sum changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Regenerate Wire dependency-injection bindings, then compile a
# fully static binary (no CGO) suitable for any Linux image.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/google/wire/cmd/wire@latest \
    && wire gen ./cmd/sisyphus

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/sisyphus ./cmd/sisyphus

# ─────────────────────────────────────────────────────────────────
# Stage 2 — Assemble the final image
#
# Inherits every OS package and third-party tool from the base
# image; this stage only adds the application binary and config.
# ─────────────────────────────────────────────────────────────────
ARG BASE_IMAGE=sisyphus-backend-base:latest
FROM ${BASE_IMAGE} AS runtime

LABEL org.opencontainers.image.title="sisyphus-backend" \
      org.opencontainers.image.description="Sisyphus DBA Platform — Backend Service" \
      org.opencontainers.image.licenses="MIT"

# ── Application binary ────────────────────────────────────────────
RUN mkdir -p /app/configs \
    && chown -R sisyphus:sisyphus /app

COPY --from=builder --chown=sisyphus:sisyphus /out/sisyphus /app/sisyphus

# ── Default application config ────────────────────────────────────
# Override at runtime via a k8s ConfigMap volume or
# `docker run -v ./config.yaml:/app/configs/config.yaml`.
COPY --chown=sisyphus:sisyphus configs/config.yaml /app/configs/config.yaml

# ── goInception default config ────────────────────────────────────
# Mount a k8s ConfigMap over /opt/goInception/config/config.toml
# to customise without rebuilding this image.
COPY --chown=sisyphus:sisyphus <<'EOF' /opt/goInception/config/config.toml
[inception]
addr            = "0.0.0.0"
port            = 4000
backup_host     = ""
backup_port     = 3306
backup_user     = ""
backup_password = ""
backup_db       = "inception"
support_charset      = "utf8,utf8mb4"
enable_nullable      = true
check_insert_field   = false
check_dml_where      = true
check_dml_limit      = true
check_dml_orderby    = true
enable_drop_table    = true
enable_drop_database = false
EOF

USER sisyphus
WORKDIR /app

EXPOSE 8000 9000

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8000/health 2>/dev/null || exit 1

ENTRYPOINT ["./sisyphus"]
CMD ["-conf", "configs/config.yaml"]
