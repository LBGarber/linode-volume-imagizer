#!/bin/ash

# <UDF name="volume_filepath" label="volume_filepath" />
# <UDF name="image_uploadurl" label="image_uploadurl" />

apk add curl

mkdir -p /build

dd if="$VOLUME_FILEPATH" of="/build/image.img" bs=1M

e2fsck -f /build/image.img
resize2fs -M /build/image.img

gzip /build/image.img

curl -v \
  -H "Content-Type: application/octet-stream" \
  --upload-file "/build/image.img.gz" \
  $IMAGE_UPLOADURL \
  --progress-bar \
  --output /dev/null