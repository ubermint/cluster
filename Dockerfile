FROM golang:1.20-alpine

WORKDIR /usr/app

COPY . .
RUN go build

CMD ["./cluster", "--master"]