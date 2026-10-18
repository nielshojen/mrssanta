# Push to another registry with e.g. make REGISTRY=<registry from terraform output>
REGISTRY ?= europe-west3-docker.pkg.dev/workplace-f488/mrssanta-repository
VERSION ?= latest

all: build

build:
	@docker buildx build --platform linux/amd64 --no-cache -t $(REGISTRY)/xsrf:$(VERSION) .
	@docker push $(REGISTRY)/xsrf:$(VERSION)
