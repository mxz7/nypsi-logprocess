ARG GO_VERSION=1.23.4
FROM golang:${GO_VERSION}-alpine as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-app .


FROM debian:bookworm-slim

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]
