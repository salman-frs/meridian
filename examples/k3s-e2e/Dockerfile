FROM golang:1.25.0-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY examples/k3s-e2e ./examples/k3s-e2e

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/meridian-k3s-e2e ./examples/k3s-e2e/cmd/meridian-k3s-e2e

FROM alpine:3.22

RUN adduser -D -u 10001 meridian

COPY --from=build /out/meridian-k3s-e2e /usr/local/bin/meridian-k3s-e2e

USER 10001

ENTRYPOINT ["/usr/local/bin/meridian-k3s-e2e"]
