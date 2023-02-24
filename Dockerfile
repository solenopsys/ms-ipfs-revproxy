# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM  golang:1.18.3-buster

ARG TARGETARCH

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./



RUN GOOS=linux GOARCH=$TARGETARCH go build  -o /go-binary

CMD [ "/go-binary" ]