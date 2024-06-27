# Use the official golang image as the base
FROM golang:alpine AS builder

# Set working directory for the build context
WORKDIR /app

# Copy the application code
COPY . .

# Install dependencies
RUN go mod download

# Build the Go binary (replace "cmd/myapp" with your actual entrypoint)
RUN go build -o einsbym-storage main.go

# Define a slimmer runtime image
FROM alpine:latest AS runtime

# Copy the built binary from the builder stage
COPY --from=builder /app/einsbym-storage /app/einsbym-storage

# Expose the port your application listens on (replace 8080 with your actual port)
EXPOSE 8080

# Set the default command to run the application
CMD ["/app/einsbym-storage"]
