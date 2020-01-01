FROM alpine:latest
RUN apk --no-cache add curl openssh-client ca-certificates tzdata

# Set local time (for cronjob sense)
RUN cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime && \
echo "Europe/Berlin" > /etc/timezone

ADD bender /app/
ADD last_docker_push /app/
WORKDIR /app
ENTRYPOINT ["/app/bender"]