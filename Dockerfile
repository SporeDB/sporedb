FROM ubuntu:16.10
ARG builddeps="git autoconf automake libtool curl make g++ unzip"
ARG rundeps="libsnappy-dev zlib1g-dev libbz2-dev"

# Install RocksDB

RUN cd /tmp && \
  apt-get update && \
  apt-get install -y $builddeps $rundeps && \
  git clone --depth 1 --branch 5.6.fb https://github.com/facebook/rocksdb.git && \
  cd rocksdb && \
  PORTABLE=1 make shared_lib && \
  INSTALL_PATH=/usr make install-shared && \
  rm -rf /tmp/rocksdb && \
  apt-get remove --purge --auto-remove -y $builddeps

RUN mkdir /sporedb
WORKDIR /sporedb

COPY sporedb /bin/sporedb
ENTRYPOINT ["sporedb"]
CMD ["-h"]
