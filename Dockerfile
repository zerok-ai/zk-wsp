# Dockerfile

# Use Alpine as the base image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /zk

# Copy the Go executable and config file
ARG APP_FILE
COPY ${APP_FILE} /zk/zk-wsp-server

# Set executable permissions for the Go executable
RUN chmod +x /zk/zk-wsp-server

# Run the Go executable
CMD ["./zk-wsp-server","-c","/zk/config/config.yaml"]