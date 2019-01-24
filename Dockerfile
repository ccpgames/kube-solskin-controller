FROM golang:alpine as builder

RUN apk add --update git

ARG PROJECT="github.com/celestialorb/solskin"
RUN mkdir -p /go/src/${PROJECT}
WORKDIR /go/src/${PROJECT}
COPY ./ ./
RUN go get ./...
RUN GOOS=linux go build -o /app ./

FROM golang:alpine
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app
ENTRYPOINT ["/app"]