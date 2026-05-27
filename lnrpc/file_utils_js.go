//go:build js

package lnrpc

import "os"

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
