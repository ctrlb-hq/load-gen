FROM golang:1.23.4-alpine AS builder

# Set environment variables
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Create app directory and copy files
WORKDIR /app
COPY . .

# Download dependencies and build the application
RUN go mod tidy \
    && go build -o log-generator

# Final minimal image
FROM alpine:latest

# Set working directory and copy binary from builder stage
WORKDIR /root/
COPY --from=builder /app/log-generator .

# Expose any ports if required (optional, e.g., EXPOSE 8080)

# Run the binary
CMD ["./log-generator"]
