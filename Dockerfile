# Buid Container
FROM golang:1.14 as builder
WORKDIR /go/src/github.com/tumf/counterblock-cache

COPY . .
# Set Environment Variable
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
# Build
RUN go get
RUN go build -o app main.go

# Runtime Container
FROM alpine

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/tumf/counterblock-cache/app /app
EXPOSE 3222
CMD ["/app"]
