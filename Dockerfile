FROM golang:1.20-bullseye as builder

WORKDIR /app

COPY . ./

RUN go build -o /app/mysqlclonedb main.go


FROM mysql:8.0.34-debian AS final

COPY --from=builder /app/mysqlclonedb /app/mysqlclonedb

CMD ["/app/mysqlclonedb"]