# Stage 1: Build the Go application
FROM alpine:latest AS builder
WORKDIR /app

# Install Go and build dependencies
RUN apk add  go gcc musl-dev sqlite-dev

# Set environment variables for Go
ENV GO111MODULE=on
# Enable CGO for SQLite
ENV CGO_ENABLED=1

# Copy go mod and sum files first to leverage Docker cache
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application (with CGO enabled for SQLite)
RUN go build -v -o main . \
    && ls -l /app/main # Keep check to verify binary exists

# Stage 2: Create the final image using Alpine
FROM alpine:latest
WORKDIR /app

# Install SQLite and other necessary packages
RUN apk add --no-cache sqlite ca-certificates curl shadow

# Create app user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Create necessary directories with proper permissions
RUN mkdir -p /app/data /app/keys && \
    chown -R appuser:appgroup /app && \
    chmod 755 /app/data /app/keys

# Copy the binary from the builder stage
COPY --from=builder /app/main .
# Copy templates and static files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Set proper ownership for all files
RUN chown -R appuser:appgroup /app && \
    chmod 755 /app/main

# Switch to non-root user
USER appuser

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application (using absolute path)
CMD ["/app/main"]
