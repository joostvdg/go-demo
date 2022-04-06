FROM golang:1.17 AS build
WORKDIR /go/src/go-demo
ARG TARGETARCH
ARG TARGETOS
COPY go.* ./
RUN go mod download
COPY . ./
RUN go build -o ./bin/go-demo main.go

FROM alpine:3
RUN apk --no-cache add ca-certificates
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/usr/bin/go-demo"]
COPY --from=build /go/src/go-demo/bin/go-demo /usr/bin/go-demo