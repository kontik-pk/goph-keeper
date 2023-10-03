FROM golang:1.21.1-alpine3.18 as builder

COPY . /app
WORKDIR /app/
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/goph-keeper github.com/kontik-pk/goph-keeper

FROM alpine
RUN apk add --no-cache ca-certificates && update-ca-certificates
COPY --from=builder /app/build/goph-keeper /usr/bin/goph-keeper
EXPOSE 8080 8080
ENTRYPOINT ["/usr/bin/goph-keeper", "run"]