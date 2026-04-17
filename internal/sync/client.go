package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os/exec"
	"strings"
	"time"

	"atbuy/noteui/internal/config"
)

type Client interface {
	PullIndex(context.Context, config.SyncProfile, PullIndexRequest) (PullIndexResponse, error)
	FetchNote(context.Context, config.SyncProfile, FetchNoteRequest) (FetchNoteResponse, error)
	RegisterNote(context.Context, config.SyncProfile, RegisterNoteRequest) (RegisterNoteResponse, error)
	PushNote(context.Context, config.SyncProfile, PushNoteRequest) (PushNoteResponse, error)
	UpdateNotePath(context.Context, config.SyncProfile, UpdateNotePathRequest) (UpdateNotePathResponse, error)
	DeleteNote(context.Context, config.SyncProfile, DeleteNoteRequest) (DeleteNoteResponse, error)
	PinsGet(context.Context, config.SyncProfile, PinsGetRequest) (PinsGetResponse, error)
	PinsPut(context.Context, config.SyncProfile, PinsPutRequest) (PinsPutResponse, error)
}

func NewClient(profile config.SyncProfile) Client {
	if config.ResolvedKind(profile) == config.SyncKindWebDAV {
		return WebDAVClient{
			HTTP:     newWebDAVHTTPClient(),
			dirCache: newWebDAVDirCache(),
		}
	}
	return SSHClient{}
}

// newWebDAVHTTPClient returns the default HTTP client used by WebDAVClient.
//
// Two reasons to not use http.DefaultClient:
//
//  1. Cookie jar. Nextcloud's session middleware sets nc_session_id on the
//     first hit and rejects follow-ups that arrive without it ("Strict cookie
//     not set"). A shared jar replays the cookie across requests and across
//     302/307 redirects inside a single call.
//  2. Timeouts. Sync often runs over a VPN where a TCP connection can silently
//     stall. The default client has no deadline and would hang forever; the
//     layered timeouts below cap dial, TLS handshake, waiting for response
//     headers, and the overall request.
func newWebDAVHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar:     jar,
		Timeout: 2 * time.Minute,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          16,
			MaxIdleConnsPerHost:   4,
			ForceAttemptHTTP2:     true,
		},
	}
}

type Runner func(context.Context, []byte, string, ...string) ([]byte, error)

type SSHClient struct{ Run Runner }

func (c SSHClient) PullIndex(ctx context.Context, profile config.SyncProfile, req PullIndexRequest) (PullIndexResponse, error) {
	var resp PullIndexResponse
	return resp, c.call(ctx, profile, "pull_index", req, &resp)
}

func (c SSHClient) FetchNote(ctx context.Context, profile config.SyncProfile, req FetchNoteRequest) (FetchNoteResponse, error) {
	var resp FetchNoteResponse
	return resp, c.call(ctx, profile, "fetch_note", req, &resp)
}

func (c SSHClient) RegisterNote(ctx context.Context, profile config.SyncProfile, req RegisterNoteRequest) (RegisterNoteResponse, error) {
	var resp RegisterNoteResponse
	return resp, c.call(ctx, profile, "register_note", req, &resp)
}

func (c SSHClient) PushNote(ctx context.Context, profile config.SyncProfile, req PushNoteRequest) (PushNoteResponse, error) {
	var resp PushNoteResponse
	return resp, c.call(ctx, profile, "push_note", req, &resp)
}

func (c SSHClient) UpdateNotePath(ctx context.Context, profile config.SyncProfile, req UpdateNotePathRequest) (UpdateNotePathResponse, error) {
	var resp UpdateNotePathResponse
	return resp, c.call(ctx, profile, "update_note_path", req, &resp)
}

func (c SSHClient) DeleteNote(ctx context.Context, profile config.SyncProfile, req DeleteNoteRequest) (DeleteNoteResponse, error) {
	var resp DeleteNoteResponse
	err := c.call(ctx, profile, "delete_note", req, &resp)
	if err == nil {
		return resp, nil
	}
	if !isUnknownOperationError(err) {
		return resp, err
	}
	return resp, c.legacyDeleteNote(ctx, profile, req)
}

func (c SSHClient) PinsGet(ctx context.Context, profile config.SyncProfile, req PinsGetRequest) (PinsGetResponse, error) {
	var resp PinsGetResponse
	return resp, c.call(ctx, profile, "pins_get", req, &resp)
}

func (c SSHClient) PinsPut(ctx context.Context, profile config.SyncProfile, req PinsPutRequest) (PinsPutResponse, error) {
	var resp PinsPutResponse
	return resp, c.call(ctx, profile, "pins_put", req, &resp)
}

func (c SSHClient) call(ctx context.Context, profile config.SyncProfile, op string, req any, out any) error {
	run := c.Run
	if run == nil {
		run = defaultRunner
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	remoteBin := profile.RemoteBin
	if remoteBin == "" {
		remoteBin = DefaultRemoteBin
	}
	data, err := run(ctx, payload, "ssh", profile.SSHHost, remoteBin, op)
	if err != nil {
		if shouldRetryLegacyRPC(err) {
			data, err = run(ctx, nil, "ssh", profile.SSHHost, remoteBin, op, string(payload))
		}
		if err != nil {
			return err
		}
	}
	var rpcResp struct {
		Error *RPCError       `json:"error"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return fmt.Errorf("decoding sync response: %w", err)
	}
	if rpcResp.Error != nil {
		return rpcResp.Error
	}
	if out == nil || len(rpcResp.Data) == 0 {
		return nil
	}
	return json.Unmarshal(rpcResp.Data, out)
}

func (c SSHClient) legacyDeleteNote(ctx context.Context, profile config.SyncProfile, req DeleteNoteRequest) error {
	run := c.Run
	if run == nil {
		run = defaultRunner
	}
	pinsResp, err := c.PinsGet(ctx, profile, PinsGetRequest{RemoteRoot: req.RemoteRoot})
	if err != nil && !isUnknownOperationError(err) {
		return err
	}
	pins := pinsResp.Pins
	pins.PinnedNoteIDs = removePinnedNoteID(pins.PinnedNoteIDs, req.NoteID)
	if _, err := c.PinsPut(ctx, profile, PinsPutRequest{RemoteRoot: req.RemoteRoot, Pins: pins}); err != nil && !isUnknownOperationError(err) {
		return err
	}
	metaPath := shellQuote(req.RemoteRoot + "/notes/" + req.NoteID + ".json")
	contentPath := shellQuote(req.RemoteRoot + "/content/" + req.NoteID + ".note")
	cmd := "rm -f " + metaPath + " " + contentPath
	_, err = run(ctx, nil, "ssh", profile.SSHHost, "sh", "-lc", cmd)
	return err
}

func isUnknownOperationError(err error) bool {
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.Code == ErrCodeInvalid && strings.TrimSpace(rpcErr.Message) == "unknown operation"
	}
	return false
}

func removePinnedNoteID(items []string, target string) []string {
	target = strings.TrimSpace(target)
	if target == "" || len(items) == 0 {
		return items
	}
	out := items[:0]
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func defaultRunner(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, stderr.String())
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func shouldRetryLegacyRPC(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "usage: noteui-sync <operation> <json-payload>")
}
