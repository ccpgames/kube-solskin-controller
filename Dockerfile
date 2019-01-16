FROM golang:onbuild
RUN mkdir -p /tmp/build
WORKDIR /tmp/build
COPY source/*.go .
RUN go get ./...
RUN GOOS=linux; go build -o app .

FROM scratch
ADD main /
CMD ["/main"]