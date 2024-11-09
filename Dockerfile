FROM golang:1.23.3 AS build

WORKDIR /build
COPY src/    /build/src
COPY main.go /build/main.go
COPY go.mod  /build/go.mod
COPY go.sum  /build/go.sum

RUN go mod download
RUN CGO_ENABLED=0 go build -o /build/bin/ur -ldflags "-s -w"

FROM gcr.io/distroless/static-debian12
COPY --from=build /build/bin/ur /app/ur

ENTRYPOINT [ "/app/ur" ]
CMD [ "" ]