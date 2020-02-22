FROM golang:1.13-alpine as basic

ENV CGO_ENABLED=0

WORKDIR /go/src/prog/
COPY . /go/src/prog/
RUN go install

FROM scratch
WORKDIR /go/bin/
COPY --from=basic /go/bin/prog ./
COPY ./content.txt /etc/content.txt

CMD ["./prog", "--content-file", "/etc/content.txt", "--logdir", "/var/log/prog"]
