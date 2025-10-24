FROM alpine:latest

RUN apk add --no-cache go make git

WORKDIR /app

COPY . .

RUN make build

CMD ["./wserver"]