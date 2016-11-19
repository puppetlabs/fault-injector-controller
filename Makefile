IMAGE_NAME = puppetlabs/fault-injector-controller
TAG = latest

build-controller :
	GOOS=linux go build -o bin/controller github.com/puppetlabs/fault-injector-controller/cmd/controller

build-controller-image : build-controller
	docker build -t $(IMAGE_NAME):$(TAG) .
