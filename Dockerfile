# Stage 1: Build the Go application
FROM golang:1.20 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Download all dependencies.
RUN go mod download

# Build the Go app
RUN go build -o main .

#############################################################################

# Stage 2: Create the final image with Ubuntu
FROM ubuntu:22.04

# Install necessary packages
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Set the user to appuser
USER appuser

# Set the working directory
WORKDIR /home/appuser

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8086

# Command to run the executable
CMD ["./main"]
