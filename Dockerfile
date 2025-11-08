FROM golang:1.25-alpine3.21 AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o webhook -ldflags '-w -extldflags "-static"' .

FROM alpine:3.21.3

RUN apk add --no-cache ca-certificates \
    && addgroup -S webhook -g 10101 \
    && adduser -S webhook -G webhook -u 10101

COPY --from=build /workspace/webhook /usr/local/bin/webhook

USER 10101:10101

ENTRYPOINT ["webhook"]
