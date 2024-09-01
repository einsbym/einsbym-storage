# Stage 1: Build
FROM golang:1.20-alpine AS builder

# Set the working directory inside the builder container
WORKDIR /app

# Copy the Go modules manifests and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Stage 2: Run
FROM alpine:3.18

# Set the working directory inside the final container
WORKDIR /app

# Copy only the necessary files from the builder stage
COPY --from=builder /app/main .
COPY splash_screen.txt .
COPY .env .

# Expose the port the app runs on
EXPOSE 8080

# Specify the command to run the Go application
CMD ["./main"]
