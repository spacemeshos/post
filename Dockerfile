FROM ubuntu:18.04 AS linux
ENV DEBIAN_FRONTEND noninteractive
ENV SHELL /bin/bash
ARG TZ=US/Eastern
ENV TZ $TZ
USER root
RUN bash -c "for i in {1..9}; do mkdir -p /usr/share/man/man\$i; done" \
 && echo 'APT::Get::Assume-Yes "true";' > /etc/apt/apt.conf.d/90noninteractive \
 && echo 'DPkg::Options "--force-confnew";' >> /etc/apt/apt.conf.d/90noninteractive \
 && apt-get update --fix-missing \
 && apt-get install -qy --no-install-recommends \
    apt-transport-https \
    ca-certificates \
    tzdata \
    locales \
    # -- trubleshuting tookit ---
    # bash \
    # curl \
    # procps \
    # net-tools \
    # -- it allows to start with nvidia-docker runtime --
    # libnvidia-compute-390 \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* \
 && locale-gen en_US.UTF-8 \
 && update-locale LANG=en_US.UTF-8 \
 && echo "$TZ" > /etc/timezone
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US.UTF-8
ENV LC_ALL en_US.UTF-8
ENV NVIDIA_REQUIRE_CUDA "cuda>=9.1 driver>=390"
ENV NVIDIA_VISIBLE_DEVICES all
ENV NVIDIA_DRIVER_CAPABILITIES compute,utility,display
LABEL com.nvidia.volumes.needed="nvidia_driver"

FROM linux as golang
ENV GOLANG_VERSION 1.15.3
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN set -ex \
 && apt-get update --fix-missing \
 && apt-get install -qy --no-install-recommends \
    git \
    bash \
    curl \
    sudo \
    unzip \
    make \
    gcc \
	libc6-dev \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* \
 && curl -L https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz | tar zx -C /usr/local \
 && go version \
 && mkdir -p "$GOPATH/src" "$GOPATH/bin" \
 && chmod -R 777 "$GOPATH"

FROM golang as server_builder
WORKDIR /go/src/github.com/spacemeshos

# Force the go compiler to use modules
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org

COPY . .

# And compile the project
RUN make build

#In this last stage, we start from a fresh Alpine image, to reduce the image size and not ship the Go compiler in our production artifacts.
FROM linux AS spacemesh

# Create an unprivileged user
RUN useradd spacemesh

# Finally we copy the statically compiled Go binary.
COPY --from=server_builder /go/src/github.com/spacemeshos/build/post /bin/
#COPY --from=server_builder /go/src/github.com/spacemeshos/build/spacemesh-init /bin/
COPY --from=server_builder /go/src/github.com/spacemeshos/build/libgpu-setup.so /bin/

# Run as an unprivileged user in its home directory by default
USER spacemesh
WORKDIR /home/spacemesh/

CMD ["/bin/post"]
