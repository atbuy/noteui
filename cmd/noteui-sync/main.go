package main

import (
	"fmt"
	"io"
	"os"

	notesync "atbuy/noteui/internal/sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: noteui-sync <operation> [json-payload-via-argv-or-stdin]")
		os.Exit(2)
	}

	var payload []byte
	if len(os.Args) >= 3 {
		payload = []byte(os.Args[2])
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			os.Exit(1)
		}
		payload = data
	}

	out, err := notesync.HandleRPC(os.Args[1], payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync rpc error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
}
