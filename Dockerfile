FROM arm32v7/alpine:latest
RUN apk --no-cache add curl openssh-client ca-certificates tzdata

# Set local time (for cronjob sense)
RUN cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime && \
echo "Europe/Berlin" > /etc/timezone && apk del tzdata

RUN mkdir /app
ADD bender /app/

ENTRYPOINT ["/app/bender"]
