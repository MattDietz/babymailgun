FROM golang:1.8-alpine3.6

WORKDIR /worker

RUN apk update && apk upgrade && apk --update add openssl git

RUN export DOCKERIZE_VERSION=v0.3.0 && wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
        && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

ADD . /worker
RUN mkdir -p /go/src/github.com/cerberus98
RUN ln -s /worker /go/src/github.com/cerberus98/babymailgun
RUN go get github.com/spf13/viper
RUN go get gopkg.in/mgo.v2
RUN go get gopkg.in/mgo.v2/bson
RUN go build cmd/worker.go

CMD ["./worker"]

