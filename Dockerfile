FROM golang:1.19-alpine

RUN apk add --no-cache build-base gcc musl-dev sqlite-dev git

WORKDIR /app


RUN go install github.com/beego/bee/v2@latest

COPY . .


ENV CGO_ENABLED=1

RUN go mod tidy

EXPOSE 8080

CMD ["bee", "run"]