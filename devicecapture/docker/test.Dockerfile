FROM golang:1.24.3 AS build
WORKDIR /usr/src/app
COPY . .

CMD ["go", "test", "-timeout", "30s", "-v", "./..."]