#!/bin/sh
apk --no-cache add curl openssh-client ca-certificates tzdata
cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime
echo "Europe/Berlin" > /etc/timezone
mkdir ~/.ssh/
echo "$CTX_SERVER_DEPLOY_SECRET" | base64 -d > ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa
./tmp/bender
