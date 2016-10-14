FROM iron/base
WORKDIR /app
COPY bender /app/
ENTRYPOINT ["./bender"]