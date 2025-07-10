# syntax=docker/dockerfile:1

# Set the default environment to production
ARG ENV=production

# ======== Base Stage ========
FROM golang:1.23-alpine AS base

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# ======== Test Stage ========
FROM base AS tester

# Copy the source code
COPY . .

# Run tests and ensure failure if any test fails
RUN go test ./... || exit 1

# Create a dummy file if tests pass
RUN echo "Tests passed" > /app/test-passed

# ======== Build Stage ========
FROM base AS builder

# Copy the source code
COPY . .

# Copy the dummy file from the test stage to enforce dependency
COPY --from=tester /app/test-passed .

# Build the Go app
RUN go build -o app ./cmd/main.go

# ======== Run Stage for Production ========
FROM alpine:latest AS production
WORKDIR /app

RUN apk add --no-cache ca-certificates

# Copy the pre-built binary file from the builder stage
COPY --from=builder /app/app .

# Ensure the app binary is executable
RUN chmod +x ./app

# Command to run the app
CMD ["./app"]

# ======== Run Stage for Development ========
FROM alpine:latest AS development
WORKDIR /app

RUN apk add --no-cache ca-certificates

# Add tzdata for timezone support
RUN apk add --no-cache tzdata

# Copy the pre-built binary file from the builder stage
COPY --from=builder /app/app .

# Ensure the app binary is executable
RUN chmod +x ./app

# Command to run the app
CMD ["./app"]

# ======== Final Stage ========
FROM ${ENV}
