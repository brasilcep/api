FROM alpine:latest AS base
RUN apk update && apk upgrade

FROM base AS builder 

RUN apk add go make git

WORKDIR /app

COPY . .

RUN make build

FROM base AS final

WORKDIR /app

COPY --from=builder /app/data.tar.gz /app/data.tar.gz

RUN tar -xzf data.tar.gz && rm -rf data.tar.gz

COPY --from=builder /app/wserver /app/wserver

CMD ["./wserver"]
