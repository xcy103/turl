COMMIT_ID				:= $(shell git rev-parse --short HEAD)
# Fall back to the short commit hash when no git tag is reachable (e.g. on a
# fork that did not inherit the upstream tags), so the image tag is never empty.
TAG_VERSION				:= $(shell git describe --abbrev=0 --tags 2>/dev/null || git rev-parse --short HEAD)
BUILD_TIME				:= $(shell date)
ARCH					:= $(shell uname -m)
# Container image repository. Override to publish under your own namespace.
IMAGE_REPO				?= beihai0xff/turl

# Pinned code-generation tool versions (single source of truth; keep tools/tools.go
# in sync). Pinning avoids @latest drift, e.g. a newer mockery requiring a newer Go.
SWAG_VERSION			:= v1.16.4
MOCKERY_VERSION			:= v2.53.6
GOMODIFYTAGS_VERSION	:= v1.17.0

# fill the ldflags with the build info
ldflags					=  "-w -X 'github.com/beihai0xff/turl/cli.version=$(TAG_VERSION)' -X 'github.com/beihai0xff/turl/cli.gitHash=$(COMMIT_ID)' -X 'github.com/beihai0xff/turl/cli.buildTime=$(BUILD_TIME)'"
BUILD_PLATFORMS 		=  linux/amd64,linux/arm64
GO_VERSION 				=  1.22-bookworm

# different Linux(MacOS) distro use different arch name, so we unify them using the same name aarch64
# eg. on MacOS with Apple silicon arch name is arm64, we use aarch64 as the arch name
ifneq ($(ARCH),x86_64)
    ARCH = aarch64
endif

# Tool installation targets. swag is the only generator the production binary
# needs (swagger docs are compiled in); mockery/gomodifytags are dev/test only.
install/swag:
	go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)

install/mockery:
	go install github.com/vektra/mockery/v2@$(MOCKERY_VERSION)

install/gomodifytags:
	go install github.com/fatih/gomodifytags@$(GOMODIFYTAGS_VERSION)

install/tools: install/swag install/mockery install/gomodifytags

bootstrap: install/tools
	go mod download -x
	make gen/swagger
	make gen/mock


lint: gen/swagger
	swag fmt -d app/turl
	golangci-lint run -v

gen/mock:
	mockery

gen/struct_tag:
	bash -x scripts/gen_configs_struct_tag.sh

gen/swagger:
	swag init --parseDependency --parseDepth 1 -d app/turl -g http.go -o docs/swagger

test: bootstrap
	docker compose -f ./internal/tests/docker-compose.yaml up -d --wait
	go test -gcflags="all=-l" -race -coverprofile=coverage.out -v ./...
	docker compose -f ./internal/tests/docker-compose.yaml down


.PHONY: bootstrap lint gen/mock gen/struct_tag gen/swagger test
.PHONY: install/swag install/mockery install/gomodifytags install/tools
#
# build section
#

build: build/docker

# build binary file. The production binary only needs the compiled-in swagger
# docs, so it installs swag and generates them — but it does NOT regenerate the
# test-only mocks (which would pull mockery and a newer Go toolchain).
build/binary: clean install/swag
	go mod download
	make gen/swagger
	go build -tags=jsoniter -ldflags=$(ldflags) -o ./build/dist/binary/turl cmd/turl/main.go

# docker: enable containerd for pulling and storing images
build/docker:
	DOCKER_BUILDKIT=1 docker buildx build \
		--ulimit nofile=1048576:1048576 \
		-f ./build/Dockerfile \
 		--build-arg BUILD_DATE="$(BUILD_TIME)" \
 		--build-arg BUILD_COMMIT="$(COMMIT_ID)" \
 		--build-arg BUILD_VERSION="$(TAG_VERSION)" \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--platform=$(BUILD_PLATFORMS) \
		--output type=docker \
		-t $(IMAGE_REPO):$(TAG_VERSION) -t $(IMAGE_REPO):latest .

build/docker_and_push:
	DOCKER_BUILDKIT=1 docker buildx build \
		--ulimit nofile=1048576:1048576 \
		-f ./build/Dockerfile \
		--build-arg BUILD_DATE="$(BUILD_TIME)" \
		--build-arg BUILD_COMMIT="$(COMMIT_ID)" \
		--build-arg BUILD_VERSION="$(TAG_VERSION)" \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--platform=$(BUILD_PLATFORMS) \
		--push \
		-t $(IMAGE_REPO):$(TAG_VERSION) -t $(IMAGE_REPO):latest .

.PHONY: build build/binary build/docker build/docker_and_push


.PHONY: clean
clean:
	rm -rf ./build/dist ./coverage.out internal/tests/mocks

#
# upload section
#


upload/docker:
	docker push

.PHONY: upload/docker



.PHONY: deploy
deploy:
	@echo "starting turl service containers..."
	docker compose -f ./internal/example/docker-compose.yaml \
		-p turl-service up -V --abort-on-container-exit
	@echo "turl service containers start successfully"

