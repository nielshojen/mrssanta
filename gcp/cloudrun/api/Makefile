VERSION=latest
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	@echo "Building on local architecture for testing ..."
	@go build -o server

deploy:
	@echo "Building on amd64 architecture for running in Cloud Run ..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o server
	@echo "Starting gcloud build ..."
	@gcloud builds submit --substitutions _TAG=${VERSION}