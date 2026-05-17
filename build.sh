#!/usr/bin/env bash

set -euo pipefail

### setup go

GOGO=/tmp/go-toolchain
# GOGO=../hackpad/cache

if ! [[ -d "$GOGO/go" ]]; then
  mkdir -p "$GOGO/go"
  TAG=go1.26.3-hackpad.6
  curl -sL "https://github.com/justwasm/go/releases/download/${TAG}/${TAG}.linux-amd64.tar.gz" | tar -xzC "$GOGO"
fi

export PATH="$GOGO/go/bin:$PATH"

which go

### build crush.wasm

go mod tidy

go run github.com/justwasm/boba/cmd/boba-wasm-build -o ./build/crush.wasm .

### bundle assets

curl -sL http://btwiuse.github.io/hackpad/wasm/init.wasm > ./build/init.wasm

cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./assets

bun build --compile --outfile=./build/index.html --target=browser ./assets/index.html

# RELAY=:8841 ufo pub ./build
