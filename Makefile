build:
  CGO_ENABLED=0 go build -o bin/go-demo main.go

sbom:
	cyclonedx-gomod app -json -output app.bom.json -licenses  .

docker-build:
	docker buildx build --platform linux/amd64,linux/arm64 -t go-demo --load .

docker-push:
	docker buildx build --platform linux/amd64,linux/arm64 -t joostvdgtanzu/go-demo:1.1.2 --push .

