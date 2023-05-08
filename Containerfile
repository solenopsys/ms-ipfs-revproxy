FROM --platform=$BUILDPLATFORM golang:buster

ARG TARGETARCH

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN GOOS=linux GOARCH=$TARGETARCH go build -o /go-binary  /app/cmd/main.go

CMD [ "/go-binary" ]