// Command beeep is a minimal CLI for manually testing native desktop
// notifications via github.com/gen2brain/beeep. It also compiles to
// GOOS=js GOARCH=wasm, where beeep uses the browser Notification API.
//
// Usage:
//
//	go run ./cmd/beeep [flags]
//	GOOS=js GOARCH=wasm go build -o beeep.wasm ./cmd/beeep
//
// Flags:
//
//	-title    notification title   (default: "Crush Test")
//	-message  notification body    (default: "Notification test from Crush")
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gen2brain/beeep"
)

func main() {
	title := flag.String("title", "Crush Test", "notification title")
	message := flag.String("message", "Notification test from Crush", "notification body")
	flag.Parse()

	beeep.AppName = "Crush"

	fmt.Printf("Sending native notification: title=%q message=%q\n", *title, *message)

	if err := beeep.Notify(*title, *message, nil); err != nil {
		log.Fatalf("Failed to send notification: %v", err)
	}

	// On js/wasm the browser Notification API may be asynchronous; wait gives
	// the JS event loop time to dispatch the notification before the process
	// exits. On all other platforms wait is a no-op.
	wait()

	fmt.Println("Done.")
}
