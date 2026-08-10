package config

import (
	"bytes"
	"encoding/json"
	"strings"
)

// prettyJSON returns data re-indented with two-space indentation and a
// trailing newline, or an error if data is not valid JSON.
//
// Used by atomicWriteFile to normalise the output of path-based edits
// (sjson.Set, sjson.Delete) which leave touched subtrees on a single
// line and never pretty-print the rest of the file.
func prettyJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// autoJSON returns prettyJSON(data) when path has a .json suffix and
// data is valid JSON. For any other extension, or when data is not
// valid JSON, the input is returned unchanged so callers of
// atomicWriteFile that store non-JSON blobs are not affected.
func autoJSON(path string, data []byte) []byte {
	if !strings.HasSuffix(path, ".json") {
		return data
	}
	pretty, err := prettyJSON(data)
	if err != nil {
		return data
	}
	return pretty
}

// pretty is like prettyJSON but takes and returns a string and panics
// on error. It is intended for test helpers and other call sites that
// already know the input is valid JSON.
func pretty(s string) string {
	pretty, err := prettyJSON([]byte(s))
	if err != nil {
		panic(err)
	}
	return string(pretty)
}
