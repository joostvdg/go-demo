FROM gcr.io/cloud-builders/go:1.17 AS build
WORKDIR /go/src/go-demo
ARG TARGETARCH
ARG TARGETOS
ARG GIT_COMMIT
ARG SEM_VER
COPY go.* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -v -ldflags="-X main.GitCommit=${GIT_COMMIT} -X main.SemVer=${SEM_VER}" -o ./bin/go-demo  main.go

#FROM alpine:3
FROM public.ecr.aws/docker/library/alpine:3.16
RUN apk --no-cache add ca-certificates
EXPOSE 8080
CMD ["/usr/bin/go-demo"]
COPY --from=build /go/src/go-demo/bin/go-demo /usr/bin/go-demo