FROM golang:1.7

RUN apt-get update && \
  apt-get install -y git autoconf automake libtool curl make g++ unzip libsnappy-dev zlib1g-dev libbz2-dev

# Install RocksDB

RUN cd /tmp && \
  git clone --depth 1 --branch 4.11.fb https://github.com/facebook/rocksdb.git && \
  cd rocksdb && \
  make shared_lib && \
  INSTALL_PATH=/usr make install-shared && \
  rm -rf /tmp/rocksdb

# Install Protoc

RUN cd /tmp && \
  git clone --branch 3.1.x --depth 1 https://github.com/google/protobuf.git && \
  cd protobuf && \
  ./autogen.sh && \
  ./configure --prefix=/usr && \
  make && \
  make install && \
  go get -u github.com/golang/protobuf/protoc-gen-go && \
  rm -rf /tmp/protobuf

# Install Gometalinter

RUN go get -u github.com/alecthomas/gometalinter && gometalinter --install
