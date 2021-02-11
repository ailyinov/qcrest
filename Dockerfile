FROM golang:1.15.6-alpine

RUN apk update
RUN apk add git

WORKDIR /app

COPY go.sum ./
COPY go.mod ./
COPY qctest.go ./

ENV PATH=$PATH:${GOPATH}/bin

RUN go generate && go build -o app .

CMD ["/app/app"]