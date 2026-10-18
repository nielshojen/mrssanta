# Push to another registry with e.g. make REGISTRY=<registry from terraform output>
REGISTRY ?= europe-west3-docker.pkg.dev/workplace-f488/mrssanta-repository
VERSION ?= latest

all: build

build:
	@docker buildx build --platform linux/amd64 --no-cache -t $(REGISTRY)/preflight:$(VERSION) .
	@docker push $(REGISTRY)/preflight:$(VERSION)
