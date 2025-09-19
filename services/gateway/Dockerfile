FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /out/gateway

FROM alpine:3.20
ENV GATEWAY_ADDR=:8080
ENV SEARCHD_URL=http://searchd:8090
COPY --from=builder /out/gateway /usr/local/bin/gateway
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway"]


