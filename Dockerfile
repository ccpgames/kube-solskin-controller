FROM golang:alpine as builder

RUN apk add --update git

RUN mkdir -p /go/src/solskin
WORKDIR /go/src/solskin
COPY source/*.go ./
RUN go get ./...
RUN GOOS=linux; go build -o /tmp/app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /tmp/app /app
CMD ["/app"]