build:
  CGO_ENABLED=0 go build -o bin/go-demo main.go

sbom:
	cyclonedx-gomod app -json -output app.bom.json -licenses  .