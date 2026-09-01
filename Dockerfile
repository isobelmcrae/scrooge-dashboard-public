FROM golang:1.26.3-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY static ./static
COPY templates ./templates
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/scrooge .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/scrooge /usr/local/bin/scrooge

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/scrooge"]
