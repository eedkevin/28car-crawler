FROM golang:1.9

WORKDIR /go/src/bitbucket.org/eedkevin/28car-crawler
COPY . .

RUN go-wrapper download
RUN go-wrapper install

CMD go-wrapper run -redis $redis -mongo $mongo