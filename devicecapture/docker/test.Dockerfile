FROM golang:1.24.3 AS builder
WORKDIR /usr/src/app
COPY . .
RUN go mod tidy
RUN go install gotest.tools/gotestsum@latest

FROM builder AS runner
WORKDIR /usr/src/app
#CMD ["go", "test", "-timeout", "30s", "-v", "./..."]
#CMD go run gotest.tools/gotestsum@latest -timeout 30s
CMD gotestsum --format pkgname-and-test-fails -- -timeout 30s ./...