ARG GO_BUILD_BASE=openblink-go-base:local
ARG RUNTIME_BASE=openblink-runtime-base:local

FROM ${GO_BUILD_BASE} AS build

WORKDIR /usr/src/build

COPY . .

RUN go build -v -o /out/deviceserver ./cmd/http

FROM ${RUNTIME_BASE} AS final

WORKDIR /usr/src/app

COPY --from=build /out/deviceserver ./deviceserver

CMD ["./deviceserver"]