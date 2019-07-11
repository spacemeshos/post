# Base build image
FROM golang:1.12.6-alpine3.10 AS build_base
RUN apk add bash make git curl unzip rsync libc6-compat gcc musl-dev
WORKDIR /go/src/github.com/spacemeshos

# Force the go compiler to use modules
ENV GO111MODULE=on

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download

# This image builds the go-spacemesh server
FROM build_base AS server_builder
# Here we copy the rest of the source code
COPY . .

# And compile the project
Run go build -o post .
RUN cd cmd/init ; go build -o ./spacemesh-init; cd ..

#In this last stage, we start from a fresh Alpine image, to reduce the image size and not ship the Go compiler in our production artifacts.
FROM alpine AS spacemesh

# Create an unprivileged user
RUN adduser -D spacemesh

# Finally we copy the statically compiled Go binary.
COPY --from=server_builder /go/src/github.com/spacemeshos/post /bin/post
COPY --from=server_builder /go/src/github.com/spacemeshos/cmd/init/spacemesh-init /bin/spacemesh-init

# Run as an unprivileged user in its home directory by default
USER spacemesh
WORKDIR /home/spacemesh/

CMD ["/bin/post"]
