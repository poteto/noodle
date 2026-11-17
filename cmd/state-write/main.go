package main

import (
	"fmt"
	"io"
	"os"

	"github.com/poteto/noodle/internal/filex"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: state-write destination (JSON on stdin)")
		os.Exit(2)
	}
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 16*1024*1024+1))
	if err == nil && len(data) > 16*1024*1024 {
		err = fmt.Errorf("state exceeds 16 MiB")
	}
	if err == nil {
		err = filex.WriteFileAtomic(os.Args[1], data)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
