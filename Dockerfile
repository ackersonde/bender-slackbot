FROM multiarch/alpine:arm64-latest-stable
RUN apk --no-cache add bash curl openssh-client ca-certificates tzdata

WORKDIR /app

# Set local time (for cronjob sense)
RUN cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime && \
echo "Europe/Berlin" > /etc/timezone && \
curl -OL https://raw.githubusercontent.com/jhubig/FritzBoxShell/master/fritzBoxShell.sh && \
curl -OL https://raw.githubusercontent.com/jhubig/FritzBoxShell/master/fritzBoxShellConfig.sh && \
chmod 755 fritzBoxShell.sh fritzBoxShellConfig.sh

ADD bender /app/

ENTRYPOINT ["/app/bender"]