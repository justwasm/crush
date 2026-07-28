#!/usr/bin/env bash

set -euo pipefail

### setup go

TAG=go1.27.0-go4js.1

GOGO=/tmp/go-toolchain
# GOGO=../hackpad/cache

if ! [[ -d "$GOGO/go" ]]; then
  mkdir -p "$GOGO/go"
  host=$(go env GOHOSTOS)
  arch=$(go env GOHOSTARCH)
  curl -sL "https://github.com/justwasm/go/releases/download/${TAG}/${TAG}.${host}-${arch}.min.tar.gz" | tar -xzC "$GOGO"
fi

export PATH="$GOGO/go/bin:$PATH"

which go

go version

### build crush.wasm

go mod tidy

# time go run github.com/justwasm/boba/cmd/boba-wasm-build -o ./build/crush.wasm .
time env GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w" -o ./build/crush.wasm .

### bundle assets

mkdir -p ./build/dist
if ! [[ -f ./build/dist/${TAG}.js-wasm.min.tar.gz ]]; then
  curl -sL https://github.com/justwasm/go/releases/download/${TAG}/${TAG}.js-wasm.min.tar.gz > ./build/dist/${TAG}.js-wasm.min.tar.gz
fi

curl -sL http://btwiuse.github.io/hackpad/wasm/init.wasm > ./build/init.wasm

cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./assets

cp ./assets/index.html ./build/index.html
cp ./assets/wanix.html ./build/wanix.html
cp ./assets/manifest.json ./build/manifest.json
cp ./assets/crush-icon-192.png ./build/crush-icon-192.png
cp ./assets/crush-icon-512.png ./build/crush-icon-512.png
cp ./assets/main.js ./build/main.js
cp ./assets/worker.js ./build/worker.js
cat ./assets/wasm_exec.js ./assets/wasm_exec.esm-wrapper.js > ./build/wasm_exec.esm.js

# RELAY=:8841 ufo pub ./build
