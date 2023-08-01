FROM golang:1.20-alpine as builder

RUN apk add git
WORKDIR /app

COPY . ./

RUN go build -o /app/mysqlclonedb main.go


FROM alpine:3.18.2 AS final

RUN apk update && apk add --no-cache mysql-client
COPY --from=builder /app/mysqlclonedb /app/mysqlclonedb

CMD ["/app/mysqlclonedb"]