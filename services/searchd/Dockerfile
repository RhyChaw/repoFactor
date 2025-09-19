FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /out/searchd

FROM alpine:3.20
ENV SEARCHD_ADDR=:8090
COPY --from=builder /out/searchd /usr/local/bin/searchd
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/searchd"]


