// Command notify is a standalone CLI tool for manually testing notification
// backends. It sends a single notification and exits, making it easy to verify
// that each backend works in the current terminal / OS environment.
//
// Usage:
//
//	go run ./cmd/notify [flags]
//
// Flags:
//
//	-backend   native | osc99 | osc777 | bell  (default: native)
//	-title     notification title               (default: "Crush Test")
//	-message   notification body text           (default: "Notification test from Crush")
//
// Examples:
//
//	go run ./cmd/notify
//	go run ./cmd/notify -backend osc99 -title "Hello" -message "World"
//	go run ./cmd/notify -backend bell
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/notification"
)

func main() {
	backend := flag.String("backend", "native", "notification backend: native | osc99 | osc777 | bell")
	title := flag.String("title", "Crush Test", "notification title")
	message := flag.String("message", "Notification test from Crush", "notification body text")
	flag.Parse()

	n := notification.Notification{
		Title:   *title,
		Message: *message,
	}

	var b notification.Backend
	switch *backend {
	case "native":
		b = notification.NewNativeBackend(notification.Icon)
	case "osc99":
		b = notification.NewOSCBackend(notification.Icon, true)
	case "osc777":
		b = notification.NewOSCBackend(notification.Icon, false)
	case "bell":
		b = notification.NewBellBackend()
	default:
		fmt.Fprintf(os.Stderr, "Unknown backend %q; choose native, osc99, osc777 or bell\n", *backend)
		os.Exit(1)
	}

	fmt.Printf("Sending %q notification: title=%q message=%q\n", *backend, *title, *message)

	cmd := b.Send(n)
	if cmd == nil {
		fmt.Println("Backend returned nil command (noop).")
		return
	}

	// Execute the command. OSC/bell backends return tea.Raw which writes escape
	// sequences to stdout; native backend triggers the OS notification daemon
	// directly.
	msg := cmd()
	if raw, ok := msg.(tea.RawMsg); ok {
		s, ok := raw.Msg.(string)
		if ok {
			fmt.Printf("Writing raw escape sequence (%d bytes) to stdout\n", len(s))
			fmt.Print(s)
		}
	}

	fmt.Println("\nDone.")
}
