BASE_PATH=gitlab.com/SporeDB/sporedb

install: protoc
	go get -t ./...

protoc:
	protoc --go_out=Mdb/version/version.proto=$(BASE_PATH)/db/version:. db/*.proto
	protoc --go_out=. db/version/*.proto

lint: install
	gometalinter -j 1 -t --deadline 100s \
		--dupl-threshold 100 \
		--exclude ".pb.go" \
		--exclude "Errors unhandled." \
		--disable interfacer \
		./...

