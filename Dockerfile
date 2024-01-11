FROM golang:1.21 as builder
RUN set -ex \
    && apt-get update --fix-missing \
    && apt-get install -qy --no-install-recommends \
    unzip sudo \
    ocl-icd-opencl-dev

WORKDIR /src
COPY Makefile* .
RUN make get-postrs-lib

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

# Here we copy the rest of the source code
COPY . .

# And compile the project
RUN --mount=type=cache,id=build,target=/root/.cache/go-build make build

FROM ubuntu:22.04 AS postcli
ENV DEBIAN_FRONTEND noninteractive
ENV SHELL /bin/bash
ARG TZ=Etc/UTC
ENV TZ $TZ
USER root
RUN set -ex \
    && apt-get update --fix-missing \
    && apt-get install -qy --no-install-recommends \
    ca-certificates \
    tzdata \
    locales \
    procps \
    net-tools \
    file \
    ocl-icd-libopencl1 clinfo \
    pocl-opencl-icd libpocl2 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && locale-gen en_US.UTF-8 \
    && update-locale LANG=en_US.UTF-8 \
    && echo "$TZ" > /etc/timezone
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US.UTF-8
ENV LC_ALL en_US.UTF-8

# Finally we copy the statically compiled Go binary.
COPY --from=builder /src/build/postcli /bin/
COPY --from=builder /src/build/libpost.so /bin/

ENTRYPOINT ["/bin/postcli"]
