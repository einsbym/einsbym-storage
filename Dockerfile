# Use the official Golang image as the base image
FROM golang:1.20-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Install the Go dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o main .

# Ensure the splash_screen.txt file is present
COPY splash_screen.txt .

# Specify the command to run the Go application
CMD ["./main"]

# Expose the port the app runs on
EXPOSE 8080
