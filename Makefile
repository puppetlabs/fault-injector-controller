IMAGE_REPOSITORY = gcr.io/puppet-panda-dev
VERSION = git

build : build-controller build-podkiller

build-images : build-controller-image build-podkiller-image

test : test-controller test-podkiller

push-images-gcr : push-controller-image-gcr push-podkiller-image-gcr

release : test build-images push-images-gcr

build-controller :
	CGO_ENABLED=0 GOOS=linux go build \
	-ldflags "-X github.com/puppetlabs/fault-injector-controller/version.Version=$(VERSION) \
	-X github.com/puppetlabs/fault-injector-controller/version.ImageRepo=$(IMAGE_REPOSITORY)" \
	$(GOARGS) -o bin/controller \
	github.com/puppetlabs/fault-injector-controller/cmd/controller

build-controller-image : build-controller
	docker build -t $(IMAGE_REPOSITORY)/fault-injector-controller:$(VERSION) -f controller.Dockerfile .

build-podkiller :
	CGO_ENABLED=0 GOOS=linux go build \
	-ldflags "-X github.com/puppetlabs/fault-injector-controller/version.Version=$(VERSION) \
	-X github.com/puppetlabs/fault-injector-controller/version.ImageRepo=$(IMAGE_REPOSITORY)" \
	$(GOARGS) -o bin/podkiller \
	github.com/puppetlabs/fault-injector-controller/cmd/podkiller

build-podkiller-image : build-podkiller
	docker build -t $(IMAGE_REPOSITORY)/fault-injector-podkiller:$(VERSION) -f podkiller.Dockerfile .

test-controller :
	go test \
	-ldflags "-X github.com/puppetlabs/fault-injector-controller/version.Version=$(VERSION) \
	-X github.com/puppetlabs/fault-injector-controller/version.ImageRepo=$(IMAGE_REPOSITORY)" \
	$(GOARGS) ./pkg/controller

test-podkiller :
	go test \
	-ldflags "-X github.com/puppetlabs/fault-injector-controller/version.Version=$(VERSION) \
	-X github.com/puppetlabs/fault-injector-controller/version.ImageRepo=$(IMAGE_REPOSITORY)" \
	$(GOARGS) ./pkg/podkiller

push-controller-image-gcr :
	gcloud docker -- push $(IMAGE_REPOSITORY)/fault-injector-controller:$(VERSION)

push-podkiller-image-gcr :
	gcloud docker -- push $(IMAGE_REPOSITORY)/fault-injector-podkiller:$(VERSION)