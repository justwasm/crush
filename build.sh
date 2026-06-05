#!/usr/bin/env bash

set -euo pipefail

### setup go

GOGO=/tmp/go-toolchain
# GOGO=../hackpad/cache

if ! [[ -d "$GOGO/go" ]]; then
  mkdir -p "$GOGO/go"
  TAG=go1.27.0-justwasm.9
  host=$(go env GOHOSTOS)
  arch=$(go env GOHOSTARCH)
  curl -sL "https://github.com/justwasm/go/releases/download/${TAG}/${TAG}.${host}-${arch}.min.tar.gz" | tar -xzC "$GOGO"
fi

export PATH="$GOGO/go/bin:$PATH"

which go

### build crush.wasm

go mod tidy

go run github.com/justwasm/boba/cmd/boba-wasm-build -o ./build/crush.wasm .

### bundle assets

curl -sL http://btwiuse.github.io/hackpad/wasm/init.wasm > ./build/init.wasm

cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./assets

cp ./assets/index.html ./build/index.html
cp ./assets/main.js ./build/main.js
cp ./assets/worker.js ./build/worker.js
cat ./assets/wasm_exec.js ./assets/wasm_exec.esm-wrapper.js > ./build/wasm_exec.esm.js

# RELAY=:8841 ufo pub ./build
