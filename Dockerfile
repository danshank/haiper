# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl tmux

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy templates
COPY --from=builder /app/templates ./templates

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]