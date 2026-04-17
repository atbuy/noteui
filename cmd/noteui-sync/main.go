package main

import (
	"fmt"
	"io"
	"os"

	notesync "atbuy/noteui/internal/sync"
)

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		_, _ = fmt.Fprintln(stderr, "usage: noteui-sync <operation> [json-payload-via-argv-or-stdin]")
		return 2
	}

	var payload []byte
	if len(args) >= 3 {
		payload = []byte(args[2])
	} else {
		data, err := io.ReadAll(stdin)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "read error: %v\n", err)
			return 1
		}
		payload = data
	}

	out, err := notesync.HandleRPC(args[1], payload)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "sync rpc error: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(out); err != nil {
		_, _ = fmt.Fprintf(stderr, "write error: %v\n", err)
		return 1
	}
	return 0
}
