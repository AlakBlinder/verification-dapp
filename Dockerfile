# Use official Golang image as base
FROM golang:1.23-bookworm

# Set working directory
WORKDIR /app

# Enable CGO and install required dependencies
ENV CGO_ENABLED=1
RUN apt-get update && apt-get install -y build-essential gcc libc-dev libgmp-dev

# Copy go mod and sum files
COPY go/go.mod go/go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Build the application
RUN go build -o main ./go/index.go

# Expose port 8080
EXPOSE 8080

# Run the binary
CMD ["./main"]