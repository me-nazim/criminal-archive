# syntax=docker/dockerfile:1.7

# ---------- build ----------
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache dependencies first
COPY go.mod go.sum* ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---------- runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api /app/api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api"]
