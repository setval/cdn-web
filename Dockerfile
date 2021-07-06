FROM golang:1.16.0-alpine3.13

WORKDIR /build
COPY . .
RUN go build -o /app/app .

RUN rm -rf /build

WORKDIR /app
COPY web.html /app/.

ENTRYPOINT ["./app"]
EXPOSE 8080