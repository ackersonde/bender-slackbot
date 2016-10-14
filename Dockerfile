FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go get ./...
RUN go build -o main .
CMD ["/app/main"]