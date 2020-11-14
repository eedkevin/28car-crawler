FROM golang:1.14.3-alpine AS builder
WORKDIR /go/src/bitbucket.org/eedkevin/28car-crawler
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /app

FROM alpine:3.4
WORKDIR /app
COPY --from=builder /app /app/.
CMD /app/app -redis $redis -mongo $mongo -crawler-mode $mode