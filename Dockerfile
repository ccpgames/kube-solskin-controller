FROM golang AS builder

ARG PROJECT="github.com/ccpgames/kube-solskin-controller"
RUN mkdir -p /go/src/${PROJECT}
WORKDIR /go/src/${PROJECT}
COPY ./vendor ./vendor
COPY ./common ./common
COPY ./exporter ./exporter
COPY ./metrics ./metrics
COPY ./suppressor ./suppressor
COPY ./main.go ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/app ./main.go

FROM scratch
COPY --from=builder /go/bin/app /go/bin/app
ENTRYPOINT ["/go/bin/app"]