FROM golang:1.21.6

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY api/ api/

RUN go build -o rinha ./api/*.go

CMD ["./rinha"]
