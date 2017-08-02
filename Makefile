BASE_PATH=gitlab.com/SporeDB/sporedb
CGO_FLAGS=CGO_LDFLAGS="-lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy"
REVISION=$(shell git rev-parse --short HEAD || echo )
DOCKER_TAG=registry.gitlab.com/sporedb/sporedb

OPT_VERSION=Mdb/version/version.proto=$(BASE_PATH)/db/version
OPT_DB=Mdb/spore.proto=$(BASE_PATH)/db

SED_RM_PROTO_SYNOPSIS=':a;N;$$!ba;s/\/\*.*\npackage/package/'

install:
	go get -t -tags rocksdb ./...

install-bolt:
	go get -t ./...

protoc:
	@protoc --go_out=plugins=grpc,$(OPT_VERSION),$(OPT_DB):. db/api/*.proto
	@protoc --go_out=$(OPT_VERSION):. db/*.proto
	@sed -i $(SED_RM_PROTO_SYNOPSIS) db/*.pb.go
	@protoc --go_out=. db/version/*.proto
	@sed -i $(SED_RM_PROTO_SYNOPSIS) db/version/*.pb.go
	@protoc --go_out=$(OPT_VERSION):. myc/protocol/*.proto
	@sed -i $(SED_RM_PROTO_SYNOPSIS) myc/protocol/*.pb.go

lint: install
	gometalinter -j 1 -t --deadline 1000s \
		--dupl-threshold 100 \
		--exclude ".pb.go" \
		--exclude "Errors unhandled." \
		--disable interfacer \
		./...

test: install
	@go test -cover $(BASE_PATH)/db/encoding
	@go test -cover $(BASE_PATH)/db
	@go test -cover $(BASE_PATH)/db/drivers/boltdb
	@go test -tags rocksdb -cover $(BASE_PATH)/db/drivers/rocksdb
	@go test -cover $(BASE_PATH)/db/version
	@go test -cover $(BASE_PATH)/myc
	@go test -cover $(BASE_PATH)/myc/protocol
	@go test -cover $(BASE_PATH)/myc/sec
	@go test $(BASE_PATH)/tests

image: install
	go build -ldflags "-s -w" .
	docker build -t $(DOCKER_TAG) .
	docker tag $(DOCKER_TAG) $(DOCKER_TAG):$(REVISION)
