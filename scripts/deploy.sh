#!/bin/bash

set -e

APP_NAME="tokenlive-admin"
VERSION=${1:-"latest"}

echo "=== Building $APP_NAME:$VERSION ==="

# Build Docker image
docker build -t $APP_NAME:$VERSION .
docker tag $APP_NAME:$VERSION $APP_NAME:latest

echo ""
echo "=== Build Complete ==="
echo "Image: $APP_NAME:$VERSION"
echo ""
echo "To run the application:"
echo "  docker-compose up -d"
echo ""
echo "Or manually:"
echo "  docker run -d -p 8040:8040 --name $APP_NAME $APP_NAME:$VERSION"
echo ""
echo "To push to registry:"
echo "  docker push $APP_NAME:$VERSION"
