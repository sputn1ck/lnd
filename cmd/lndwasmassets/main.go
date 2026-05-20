package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sputn1ck/go-wasmsqlite"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <dest-dir>\n", os.Args[0])
		os.Exit(2)
	}

	destDir := os.Args[1]
	if err := wasmsqlite.ExtractAssets(destDir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "extract sqlite assets: %v\n", err)
		os.Exit(1)
	}

	wasmExec := filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js")
	if err := copyFile(wasmExec, filepath.Join(destDir, "wasm_exec.js")); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "copy wasm_exec.js: %v\n", err)
		os.Exit(1)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
