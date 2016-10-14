FROM golang:latest
RUN mkdir /app
ADD . /app/
ENV GOPATH /app
WORKDIR /app
RUN go get ./...
RUN go build -o main .
CMD ["/app/main"]