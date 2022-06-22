FROM golang:1.17 AS build
WORKDIR /go/src/go-demo
ARG TARGETARCH
ARG TARGETOS
ARG GIT_COMMIT
ARG SEM_VER
COPY go.* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o ./bin/go-demo -ldflags "-X main.gitCommit=$(GIT_COMMIT) -X main.semver=${SEM_VER}" main.go

FROM alpine:3
RUN apk --no-cache add ca-certificates
EXPOSE 8080
CMD ["/usr/bin/go-demo"]
COPY --from=build /go/src/go-demo/bin/go-demo /usr/bin/go-demo