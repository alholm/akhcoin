FROM golang:alpine

WORKDIR /go/src/akhcoin
COPY . .

RUN apk add --no-cache git
RUN go-wrapper download   # "go get -d -v ./..."
RUN go-wrapper install    # "go install -v ./..."
RUN apk del git

EXPOSE 9765 8765

CMD ["go-wrapper", "run"] # ["App"]