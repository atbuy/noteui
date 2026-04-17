package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunRequiresOperation(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"noteui-sync"}, strings.NewReader(""), &stdout, &stderr)

	require.Equal(t, 2, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "usage: noteui-sync")
}

func TestRunReadsPayloadFromStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"noteui-sync", "unknown_operation"}, strings.NewReader(`{"ignored":true}`), &stdout, &stderr)

	require.Equal(t, 0, code)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), `"code":"invalid_request"`)
	require.Contains(t, stdout.String(), `"message":"unknown operation"`)
}

func TestRunUsesArgvPayloadWhenProvided(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"noteui-sync", "unknown_operation", `{"ignored":true}`}, strings.NewReader("not used"), &stdout, &stderr)

	require.Equal(t, 0, code)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), `"code":"invalid_request"`)
	require.Contains(t, stdout.String(), `"message":"unknown operation"`)
}

func TestRunReportsReadError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"noteui-sync", "unknown_operation"}, errReader{err: errors.New("boom")}, &stdout, &stderr)

	require.Equal(t, 1, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "read error: boom")
}

func TestRunReportsWriteError(t *testing.T) {
	var stderr bytes.Buffer

	code := run([]string{"noteui-sync", "unknown_operation", `{"ignored":true}`}, strings.NewReader(""), errWriter{err: errors.New("disk full")}, &stderr)

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "write error: disk full")
}

type errReader struct {
	err error
}

func (r errReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

type errWriter struct {
	err error
}

func (w errWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}
