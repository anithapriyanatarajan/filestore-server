# Use the official Golang image as a base image
FROM golang:1.17-alpine

# Set the working directory
WORKDIR /app

# Copy the server code and required files into the container
COPY . .

# Build the Go application
RUN go build -o server

# Expose the port the server will run on
EXPOSE 8080

# Command to run the server
CMD ["./server"]