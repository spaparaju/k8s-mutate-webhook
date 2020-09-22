NAME = mutateme
IMAGE_PREFIX = docker.io/meerakrishna
IMAGE_NAME = crc-mutating-webhook
IMAGE_VERSION = 7.0

export GO111MODULE=on

app: deps
	go build -v -o $(NAME) cmd/main.go

deps:
	go get -v ./...

test: deps
	go test -v ./... -cover
	
docker:
	docker build --no-cache -t $(IMAGE_PREFIX)/$(IMAGE_NAME):$(IMAGE_VERSION) .
	docker tag $(IMAGE_PREFIX)/$(IMAGE_NAME):$(IMAGE_VERSION) $(IMAGE_PREFIX)/$(IMAGE_NAME):latest

push:
	@echo "WARNING: if you push to a public repo, you're pushing ssl key & cert, are you sure? [CTRL-C to cancel, ANY other to continue]"
	@sh read -n 1
	docker push $(IMAGE_PREFIX)/$(IMAGE_NAME):$(IMAGE_VERSION) 
	docker push $(IMAGE_PREFIX)/$(IMAGE_NAME):latest

kind:
	kind create cluster --config kind.yaml

deploy:
	export KUBECONFIG=$$(kind get kubeconfig-path --name="kind"); kubectl apply -f deploy/

reset:
	export KUBECONFIG=$$(kind get kubeconfig-path --name="kind"); kubectl delete -f deploy/
	kind delete cluster --name kind

.PHONY: docker push kind deploy reset
