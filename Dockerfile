FROM golang:alpine AS builder

RUN apk add --update --no-cache git
RUN go get -u github.com/golang/dep/cmd/dep

ARG PROJECT="github.com/ccpgames/kube-solskin-controller"
RUN mkdir -p /go/src/${PROJECT}
WORKDIR /go/src/${PROJECT}
COPY ./ ./
RUN dep ensure
RUN GOOS=linux go build -o /app ./

FROM golang:alpine
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app
ENTRYPOINT ["/app"]