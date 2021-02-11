FROM golang:1.15.6-alpine

WORKDIR /app

COPY go.sum ./
COPY go.mod ./
COPY /src ./

ENV PATH=$PATH:${GOPATH}/bin

RUN go build -o app /app/main

CMD ["/app/app"]