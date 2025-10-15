FROM golang:1.24.4-alpine

WORKDIR /app


RUN go install github.com/beego/bee/v2@latest

COPY . .


RUN go mod tidy && mkdir data


EXPOSE 8080

CMD ["bee", "run"]