FROM alpine:latest

RUN apk add go make git

WORKDIR /app

COPY . .

RUN tar -xzf data.tar.gz && rm -rf data.tar.gz

RUN make build

CMD ["./wserver"]