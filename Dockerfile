# Stage 1: Build the binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a static, stripped binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/rafibot .

# Stage 2: Create a minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/rafibot /rafibot

# Run the binary
USER nonroot:nonroot
ENTRYPOINT ["/rafibot", "-c /config/config.toml"]
