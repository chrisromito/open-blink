ARG GO_BUILD_BASE=openblink-go-base:local

FROM ${GO_BUILD_BASE} AS runner

WORKDIR /usr/src/app

COPY . .

CMD ["gotestsum", "--format", "pkgname-and-test-fails", "--", "-timeout", "30s", "./..."]