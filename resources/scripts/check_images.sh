#!/bin/bash

IMAGE_URL=$1
shift
IMAGES=("$@")

for IMAGE in "${IMAGES[@]}"; do
  if ! docker manifest inspect "${IMAGE_URL}/${IMAGE}" > /dev/null 2>&1; then
    echo "Image ${IMAGE_URL}/${IMAGE} does not exist."
    exit 1
  fi
done

echo "All images exist."
exit 0
