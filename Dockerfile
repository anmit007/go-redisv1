# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Download dependencies (currently none, but good practice)
RUN go mod download

# Copy source code
COPY . .

# Build the binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o go-redis .

# Runtime stage - use minimal alpine image
FROM alpine:3.19

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/go-redis .

# Create directory for AOF persistence
RUN mkdir -p /data

# Expose the default port
EXPOSE 7379

# Run the server
CMD ["./go-redis", "-host", "0.0.0.0", "-port", "7379"]
