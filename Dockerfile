FROM alpine:latest

# install Go
RUN mkdir -p /root/gocode
ENV GOPATH /root/gocode
RUN apk add -U git go

# install bender-slackbot
RUN git clone https://github.com/danackerson/bender-slackbot.git $GOPATH/src/github.com/danackerson/bender-slackbot/
WORKDIR $GOPATH/src/github.com/danackerson/bender-slackbot

RUN go get ./...
RUN go test

RUN go build bender.go
RUN mv bender /root/

RUN apk del git go && \
  rm -rf $GOPATH/pkg && \
  rm -rf $GOPATH/bin && \
  rm -rf $GOPATH/src/gopkg.in && \
  rm -rf $GOPATH/src/github.com/nlopes && \
  rm -rf /var/cache/apk/*

# execute bender slackbot
ENTRYPOINT ["/root/bender"]
