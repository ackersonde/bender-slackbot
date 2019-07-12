#!/bin/sh
apk --no-cache add curl openssh-client ca-certificates tzdata
cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime
echo "Europe/Berlin" > /etc/timezone
./tmp/bender
