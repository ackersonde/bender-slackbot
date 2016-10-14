FROM golang:latest
RUN mkdir /app
ADD bender /app/
WORKDIR /app
RUN go build -o main 
CMD ["/app/main"]
