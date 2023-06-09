# Dockerfile

# Use Alpine as the base image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the Go executable and config file
ARG APP_FILE
ARG CONFIG_FILE

COPY ${APP_FILE} /app/myapp
COPY ${CONFIG_FILE} /app/config.yaml

# Set executable permissions for the Go executable
RUN chmod +x /app/myapp

# Run the Go executable
CMD ["./myapp","-config","config.yaml"]