#!/bin/ash

# <UDF name="volume_filepath" label="volume_filepath" />

apk add curl docker pv

mkdir -p /build

dd if="$VOLUME_FILEPATH" of="/build/image.img" bs=4M

e2fsck -f /build/image.img
resize2fs -M /build/image.img

service docker start
timeout 15 sh -c "until docker info; do echo .; sleep 1; done"

docker run --name nginx-server -p 8081:80 -v /build:/usr/share/nginx/html:ro -d nginx