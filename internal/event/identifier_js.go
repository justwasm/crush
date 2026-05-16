//go:build js && wasm

package event

import (
	"syscall/js"

	"github.com/google/uuid"
)

var distinctId string

const (
	fallbackId = "unknown"
	storageKey = "crush_distinct_id"
)

func getDistinctId() string {
	// Try reading from localStorage first.
	if id := readFromStorage(); id != "" {
		return id
	}

	// Generate a new UUID and persist it.
	if id := generateAndStore(); id != "" {
		return id
	}

	return fallbackId
}

func readFromStorage() string {
	storage := js.Global().Get("localStorage")
	if !storage.Truthy() {
		return ""
	}
	v := storage.Call("getItem", storageKey)
	if !v.Truthy() {
		return ""
	}
	return v.String()
}

func generateAndStore() string {
	id := uuid.New().String()

	storage := js.Global().Get("localStorage")
	if storage.Truthy() {
		storage.Call("setItem", storageKey, id)
	}
	return id
}
