# Stage 1: Build stage
FROM golang:1.25-alpine AS builder

# Install make
RUN apk add --no-cache make git

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Cache Go Modules
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN make build

# Stage 2: Run stage
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the binary from the build stage
COPY --from=builder /app/bin/goodiesdb-server .


# Expose port 6379 to the outside world
EXPOSE 6379

# Command to run the executable
CMD ["./goodiesdb-server"]
