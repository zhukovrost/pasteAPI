# Start with the official Go image as a base image
FROM golang:1.22-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files to the working directory
COPY go.mod go.sum ./

# Download and cache Go modules
RUN go mod download

# Copy the rest of the application code to the working directory
COPY . .

# Ensure that bash and make are installed
RUN apk add --no-cache bash make

# Build the application using the Makefile
RUN make build/api

# Create a new stage for a minimal runtime image
FROM alpine:latest

# Install any necessary dependencies (e.g., CA certificates)
RUN apk --no-cache add ca-certificates

# Set the working directory inside the container
WORKDIR /root/

# Copy the built application from the builder stage
COPY --from=builder /app/bin/api ./api

# Copy configuration files, if necessary
COPY --from=builder /app/configs/config.yml ./configs/config.yml

# Expose the port on which the application will run
EXPOSE 8080

# Run the application
CMD ["./api"]
