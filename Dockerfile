# syntax=docker/dockerfile:1

# ─────────────────────────────────────────────
# Stage 1: Build React Frontend
# ─────────────────────────────────────────────
FROM node:22-alpine AS frontend

WORKDIR /build

COPY mangastu-web/package*.json ./
RUN npm ci

COPY mangastu-web/ .
RUN npm run build


# ─────────────────────────────────────────────
# Stage 2: Build Pure Static Go Binary
# ─────────────────────────────────────────────
FROM golang:alpine AS backend

WORKDIR /build

ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 \
    GOOS=linux \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o mangastu \
    ./cmd/mangastu


# ─────────────────────────────────────────────
# Stage 3: Minimal Scratch Runtime (< 30MB total)
# ─────────────────────────────────────────────
FROM scratch

WORKDIR /app

# Copy /tmp for temporary file processing
COPY --from=backend /tmp /tmp

# Copy static Go binary
COPY --from=backend /build/mangastu /app/mangastu

# Copy React production distribution
COPY --from=frontend /build/dist /app/dist

# Copy HTTPS CA certificates
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Expose default HTTP server port
EXPOSE 8080

ENTRYPOINT ["/app/mangastu", "serve", "--port", "8080", "--static-dir", "/app/dist"]
