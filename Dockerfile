FROM golang:1.19 AS BUILDER

# install build deps
RUN apt-get update
RUN apt-get install -y protobuf-compiler
RUN export GO111MODULE=on
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN export PATH="$PATH:$(go env GOPATH)/bin"

WORKDIR /build

# build protobuf
COPY protobuf protobuf
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    protobuf/exhook.proto

# cache the module downloads
COPY go.mod .
COPY go.sum .
RUN go mod download

# copy sources and build
COPY src src
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o ./bin/emqx_grpc_hook_append_time ./src 

# final image starting from scratch!
FROM scratch

# copy only built binary
COPY --from=BUILDER /build/bin/ /

ENTRYPOINT ["/emqx_grpc_hook_append_time"]
