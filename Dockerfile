FROM golang:1.14 as build

WORKDIR /go/src/github.com/contextcloud/openfaas-cloudpubsub-connector

COPY . .

# Run a gofmt and exclude all vendored code.
RUN go test -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o /usr/bin/connector

FROM alpine:3.11 as ship
RUN apk add --no-cache ca-certificates

COPY --from=build /usr/bin/connector /usr/bin/connector
WORKDIR /root/

CMD ["/usr/bin/connector"]