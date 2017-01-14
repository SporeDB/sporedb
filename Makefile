BASE_PATH=gitlab.com/SporeDB/sporedb
CGO_FLAGS=CGO_LDFLAGS="-lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy"

OPT_VERSION=Mdb/version/version.proto=$(BASE_PATH)/db/version
OPT_DB=Mdb/spore.proto=$(BASE_PATH)/db

install: protoc
	go get -v -t ./...

protoc:
	protoc --go_out=plugins=grpc,$(OPT_VERSION),$(OPT_DB):. db/api/*.proto
	protoc --go_out=$(OPT_VERSION):. db/*.proto
	protoc --go_out=. db/version/*.proto
	protoc --go_out=. myc/protocol/*.proto

lint: install
	gometalinter -j 1 -t --deadline 1000s \
		--dupl-threshold 100 \
		--exclude ".pb.go" \
		--exclude "Errors unhandled." \
		--disable interfacer \
		./...

test: install
	@go test -cover $(BASE_PATH)/db
	@go test -cover $(BASE_PATH)/db/drivers/rocksdb
	@go test -cover $(BASE_PATH)/db/version
	@go test -cover $(BASE_PATH)/myc
	@go test -cover $(BASE_PATH)/myc/protocol

