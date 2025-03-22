#!/bin/bash

# Configuration
GOOS="linux"
GOARCH="arm64"
EXECUTABLE_NAME="spectrum-capture"  # Replace with your desired executable name
REMOTE_USER="pi"         # Replace with the remote username
REMOTE_HOST="spectrum.local"         # Replace with the remote host (IP or domain)
REMOTE_PATH="~/soapypower-container" # Replace with the directory on the remote machine where you want to copy the file

# Step 1: Build the Go application
echo "Building the Go application..."
GOOS=$GOOS GOARCH=$GOARCH go build -o $EXECUTABLE_NAME
if [ $? -ne 0 ]; then
  echo "Build failed. Exiting."
  exit 1
fi
echo "Build successful!"

# Step 2: Copy the executable to the remote destination using scp
echo "Copying the executable to the remote destination..."
scp $EXECUTABLE_NAME $REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH
if [ $? -ne 0 ]; then
  echo "Failed to copy the file to the remote destination. Exiting."
  exit 1
fi
echo "File copied successfully!"

# Step 3: Run `docker-compose up` on the remote system and stream logs
echo "Running docker-compose up on the remote destination and streaming logs..."
ssh -t $REMOTE_USER@$REMOTE_HOST "cd $REMOTE_PATH && docker compose up"
if [ $? -ne 0 ]; then
  echo "Failed to start Docker Compose on the remote system. Exiting."
  exit 1
fi

echo "Docker Compose started successfully!"
