package sync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

type WebDAVClient struct {
	HTTP     *http.Client
	dirCache *webdavDirCache
}

type webdavDirCache struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newWebDAVDirCache() *webdavDirCache {
	return &webdavDirCache{seen: make(map[string]struct{})}
}

func (c WebDAVClient) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c WebDAVClient) PullIndex(ctx context.Context, profile config.SyncProfile, req PullIndexRequest) (PullIndexResponse, error) {
	var resp PullIndexResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)
	baseEntries, err := c.propfindURL(ctx, profile, baseURL, 1)
	if err != nil {
		return resp, fmt.Errorf("webdav pull index: %w", err)
	}
	metaDirPath := strings.TrimRight(urlPath(baseURL), "/") + "/.noteui-sync"
	if !containsCollection(baseEntries, metaDirPath) {
		return resp, nil
	}

	serverBase := serverBaseURL(profile)
	entries, err := c.propfindURL(ctx, profile, strings.TrimRight(baseURL, "/")+"/.noteui-sync/notes/", 1)
	if err != nil {
		return resp, fmt.Errorf("webdav pull index: %w", err)
	}

	candidates := make([]propfindEntry, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasSuffix(entry.href, ".json") {
			continue
		}
		if !strings.Contains(entry.href, "/.noteui-sync/notes/") {
			continue
		}
		candidates = append(candidates, entry)
	}

	notesOut, skipped := c.fetchNotesConcurrent(ctx, profile, baseURL, serverBase, candidates)
	sort.Slice(notesOut, func(i, j int) bool { return notesOut[i].RelPath < notesOut[j].RelPath })
	resp.Notes = notesOut
	resp.SkippedCount = skipped

	pinsURL := baseURL + "/.noteui-sync/pins.json"
	pinsBody, _, err := c.getFile(ctx, profile, pinsURL)
	if err == nil && len(pinsBody) > 0 {
		_ = json.Unmarshal(pinsBody, &resp.Pins)
	}

	return resp, nil
}

// fetchNotesConcurrent fetches mapping + note body for each candidate entry
// using a small worker pool. Concurrency is bounded on purpose: WebDAV
// servers (Nextcloud especially) throttle hard when a single client opens
// many simultaneous connections, so a handful of workers beats a flood.
// Entries whose mapping or body fails are counted into skipped so the caller
// can surface partial failures instead of silently dropping notes.
func (c WebDAVClient) fetchNotesConcurrent(
	ctx context.Context,
	profile config.SyncProfile,
	baseURL, serverBase string,
	candidates []propfindEntry,
) ([]RemoteNoteMeta, int) {
	const workers = 4
	if len(candidates) == 0 {
		return nil, 0
	}

	jobs := make(chan propfindEntry)
	type result struct {
		note RemoteNoteMeta
		ok   bool
	}
	results := make(chan result, len(candidates))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entry := range jobs {
				if ctx.Err() != nil {
					results <- result{}
					continue
				}
				note, ok := c.fetchNote(ctx, profile, baseURL, serverBase, entry)
				results <- result{note: note, ok: ok}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, entry := range candidates {
			select {
			case <-ctx.Done():
				return
			case jobs <- entry:
			}
		}
	}()

	wg.Wait()
	close(results)

	out := make([]RemoteNoteMeta, 0, len(candidates))
	skipped := 0
	for r := range results {
		if !r.ok {
			skipped++
			continue
		}
		out = append(out, r.note)
	}
	return out, skipped
}

func (c WebDAVClient) fetchNote(
	ctx context.Context,
	profile config.SyncProfile,
	baseURL, serverBase string,
	entry propfindEntry,
) (RemoteNoteMeta, bool) {
	mappingURL := resolveHref(serverBase, entry.href)
	body, _, err := c.getFile(ctx, profile, mappingURL)
	if err != nil {
		return RemoteNoteMeta{}, false
	}
	var mapping webdavNoteMapping
	if err := json.Unmarshal(body, &mapping); err != nil {
		return RemoteNoteMeta{}, false
	}
	noteURL := baseURL + "/" + escapePath(mapping.RelPath)
	noteBody, noteEtag, err := c.getFile(ctx, profile, noteURL)
	if err != nil {
		return RemoteNoteMeta{}, false
	}
	title := notes.ExtractTitle(string(noteBody))
	if title == "" {
		title = path.Base(mapping.RelPath)
	}
	return RemoteNoteMeta{
		ID:        mapping.ID,
		RelPath:   mapping.RelPath,
		Title:     title,
		Revision:  buildWebDAVRevision(noteEtag, noteBody),
		Encrypted: mapping.Encrypted,
	}, true
}

func (c WebDAVClient) FetchNote(ctx context.Context, profile config.SyncProfile, req FetchNoteRequest) (FetchNoteResponse, error) {
	var resp FetchNoteResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)

	mapping, err := c.loadNoteMapping(ctx, profile, baseURL, req.NoteID)
	if err != nil {
		return resp, err
	}

	noteURL := baseURL + "/" + escapePath(mapping.RelPath)
	body, etag, err := c.getFile(ctx, profile, noteURL)
	if err != nil {
		return resp, fmt.Errorf("webdav fetch note content: %w", err)
	}
	rev := buildWebDAVRevision(etag, body)
	title := notes.ExtractTitle(string(body))
	if title == "" {
		title = path.Base(mapping.RelPath)
	}
	resp.Note = RemoteNote{
		RemoteNoteMeta: RemoteNoteMeta{
			ID:        mapping.ID,
			RelPath:   mapping.RelPath,
			Title:     title,
			Revision:  rev,
			Encrypted: mapping.Encrypted,
		},
		Content: string(body),
	}
	return resp, nil
}

func (c WebDAVClient) RegisterNote(ctx context.Context, profile config.SyncProfile, req RegisterNoteRequest) (RegisterNoteResponse, error) {
	var resp RegisterNoteResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)
	id := "n_" + strings.ReplaceAll(NewClientID(), "client-", "")

	noteURL := baseURL + "/" + escapePath(req.RelPath)
	c.mkcolParents(ctx, profile, baseURL, noteURL)
	etag, err := c.putFile(ctx, profile, noteURL, []byte(req.Content), "")
	if err != nil {
		return resp, fmt.Errorf("webdav register note: %w", err)
	}
	rev := buildWebDAVRevision(etag, []byte(req.Content))

	mapping := webdavNoteMapping{
		ID:        id,
		RelPath:   req.RelPath,
		Encrypted: req.Encrypted,
	}
	if err := c.saveNoteMapping(ctx, profile, baseURL, mapping); err != nil {
		return resp, err
	}

	resp.ID = id
	resp.Revision = rev
	return resp, nil
}

func (c WebDAVClient) PushNote(ctx context.Context, profile config.SyncProfile, req PushNoteRequest) (PushNoteResponse, error) {
	var resp PushNoteResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)

	mapping, err := c.loadNoteMapping(ctx, profile, baseURL, req.NoteID)
	if err != nil {
		return resp, err
	}

	noteURL := baseURL + "/" + escapePath(mapping.RelPath)
	currentBody, currentEtag, err := c.getFile(ctx, profile, noteURL)
	if err != nil {
		return resp, fmt.Errorf("webdav push: fetch current: %w", err)
	}

	currentRev := buildWebDAVRevision(currentEtag, currentBody)
	if req.ExpectedRevision != "" && !sameRevision(currentRev, req.ExpectedRevision) {
		return resp, &RPCError{Code: ErrCodeConflict, Message: "revision mismatch"}
	}

	if mapping.RelPath != req.RelPath {
		oldURL := baseURL + "/" + escapePath(mapping.RelPath)
		newURL := baseURL + "/" + escapePath(req.RelPath)
		c.mkcolParents(ctx, profile, baseURL, newURL)
		if err := c.moveFile(ctx, profile, oldURL, newURL); err != nil {
			return resp, fmt.Errorf("webdav push note move: %w", err)
		}
		mapping.RelPath = req.RelPath
	}

	noteURL = baseURL + "/" + escapePath(req.RelPath)
	c.mkcolParents(ctx, profile, baseURL, noteURL)
	etag, err := c.putFile(ctx, profile, noteURL, []byte(req.Content), "")
	if err != nil {
		return resp, fmt.Errorf("webdav push note: %w", err)
	}
	rev := buildWebDAVRevision(etag, []byte(req.Content))

	mapping.Encrypted = req.Encrypted
	if err := c.saveNoteMapping(ctx, profile, baseURL, mapping); err != nil {
		return resp, err
	}

	resp.Revision = rev
	return resp, nil
}

func (c WebDAVClient) UpdateNotePath(ctx context.Context, profile config.SyncProfile, req UpdateNotePathRequest) (UpdateNotePathResponse, error) {
	var resp UpdateNotePathResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)

	mapping, err := c.loadNoteMapping(ctx, profile, baseURL, req.NoteID)
	if err != nil {
		return resp, err
	}

	noteURL := baseURL + "/" + escapePath(mapping.RelPath)
	currentBody, currentEtag, err := c.getFile(ctx, profile, noteURL)
	if err != nil {
		return resp, fmt.Errorf("webdav update path: fetch current: %w", err)
	}
	currentRev := buildWebDAVRevision(currentEtag, currentBody)
	if req.ExpectedRevision != "" && !sameRevision(currentRev, req.ExpectedRevision) {
		return resp, &RPCError{Code: ErrCodeConflict, Message: "revision mismatch"}
	}

	oldURL := noteURL
	newURL := baseURL + "/" + escapePath(req.RelPath)
	c.mkcolParents(ctx, profile, baseURL, newURL)
	if err := c.moveFile(ctx, profile, oldURL, newURL); err != nil {
		return resp, fmt.Errorf("webdav move: %w", err)
	}

	_, newEtag, err := c.getFile(ctx, profile, newURL)
	if err != nil {
		return resp, fmt.Errorf("webdav update path: fetch new: %w", err)
	}
	rev := buildWebDAVRevision(newEtag, currentBody)

	mapping.RelPath = req.RelPath
	if err := c.saveNoteMapping(ctx, profile, baseURL, mapping); err != nil {
		return resp, err
	}

	resp.Revision = rev
	return resp, nil
}

func (c WebDAVClient) DeleteNote(ctx context.Context, profile config.SyncProfile, req DeleteNoteRequest) (DeleteNoteResponse, error) {
	var resp DeleteNoteResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)

	mapping, err := c.loadNoteMapping(ctx, profile, baseURL, req.NoteID)
	if err != nil {
		return resp, err
	}

	noteURL := baseURL + "/" + escapePath(mapping.RelPath)
	currentBody, currentEtag, err := c.getFile(ctx, profile, noteURL)
	if err != nil {
		return resp, fmt.Errorf("webdav delete note: fetch current: %w", err)
	}
	currentRev := buildWebDAVRevision(currentEtag, currentBody)
	if req.ExpectedRevision != "" && !sameRevision(currentRev, req.ExpectedRevision) {
		return resp, &RPCError{Code: ErrCodeConflict, Message: "revision mismatch"}
	}
	if err := c.deleteFile(ctx, profile, noteURL); err != nil {
		return resp, fmt.Errorf("webdav delete note content: %w", err)
	}

	mappingURL := baseURL + "/.noteui-sync/notes/" + url.PathEscape(req.NoteID) + ".json"
	if err := c.deleteFile(ctx, profile, mappingURL); err != nil {
		return resp, fmt.Errorf("webdav delete note mapping: %w", err)
	}

	pinsURL := baseURL + "/.noteui-sync/pins.json"
	pinsBody, _, err := c.getFile(ctx, profile, pinsURL)
	if err == nil && len(pinsBody) > 0 {
		var pins Pins
		if json.Unmarshal(pinsBody, &pins) == nil {
			pins.PinnedNoteIDs = removePinnedNoteID(pins.PinnedNoteIDs, req.NoteID)
			if data, err := json.MarshalIndent(pins, "", "  "); err == nil {
				if _, err := c.putFile(ctx, profile, pinsURL, data, ""); err != nil {
					return resp, fmt.Errorf("webdav delete note pins: %w", err)
				}
			}
		}
	} else if err != nil {
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != ErrCodeNotFound {
			return resp, fmt.Errorf("webdav delete note pins: %w", err)
		}
	}

	return resp, nil
}

func (c WebDAVClient) PinsGet(ctx context.Context, profile config.SyncProfile, req PinsGetRequest) (PinsGetResponse, error) {
	var resp PinsGetResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)
	pinsURL := baseURL + "/.noteui-sync/pins.json"
	body, _, err := c.getFile(ctx, profile, pinsURL)
	if err != nil {
		return resp, nil
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &resp.Pins)
	}
	return resp, nil
}

func (c WebDAVClient) PinsPut(ctx context.Context, profile config.SyncProfile, req PinsPutRequest) (PinsPutResponse, error) {
	var resp PinsPutResponse
	baseURL := webdavBaseURL(profile, req.RemoteRoot)
	pinsURL := baseURL + "/.noteui-sync/pins.json"

	data, err := json.MarshalIndent(req.Pins, "", "  ")
	if err != nil {
		return resp, err
	}
	c.mkcolParents(ctx, profile, baseURL, pinsURL)
	if _, err := c.putFile(ctx, profile, pinsURL, data, ""); err != nil {
		return resp, fmt.Errorf("webdav pins put: %w", err)
	}
	resp.Pins = req.Pins
	return resp, nil
}

// webdavNoteMapping is stored at <remote_root>/.noteui-sync/notes/<id>.json.
type webdavNoteMapping struct {
	ID        string `json:"id"`
	RelPath   string `json:"rel_path"`
	Encrypted bool   `json:"encrypted"`
}

func (c WebDAVClient) loadNoteMapping(ctx context.Context, profile config.SyncProfile, baseURL, noteID string) (webdavNoteMapping, error) {
	mappingURL := baseURL + "/.noteui-sync/notes/" + url.PathEscape(noteID) + ".json"
	body, _, err := c.getFile(ctx, profile, mappingURL)
	if err != nil {
		return webdavNoteMapping{}, &RPCError{Code: ErrCodeNotFound, Message: "note not found"}
	}
	var mapping webdavNoteMapping
	if err := json.Unmarshal(body, &mapping); err != nil {
		return webdavNoteMapping{}, fmt.Errorf("corrupt note mapping for %s: %w", noteID, err)
	}
	return mapping, nil
}

func (c WebDAVClient) saveNoteMapping(ctx context.Context, profile config.SyncProfile, baseURL string, mapping webdavNoteMapping) error {
	mappingURL := baseURL + "/.noteui-sync/notes/" + url.PathEscape(mapping.ID) + ".json"
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return err
	}
	c.mkcolParents(ctx, profile, baseURL, mappingURL)
	if _, err := c.putFile(ctx, profile, mappingURL, data, ""); err != nil {
		return fmt.Errorf("webdav save mapping: %w", err)
	}
	return nil
}

type davMultistatus struct {
	XMLName   xml.Name      `xml:"DAV: multistatus"`
	Responses []davResponse `xml:"response"`
}

type davResponse struct {
	Href     string      `xml:"href"`
	Propstat davPropstat `xml:"propstat"`
}

type davPropstat struct {
	Prop   davProp `xml:"prop"`
	Status string  `xml:"status"`
}

type davProp struct {
	ResourceType davResourceType `xml:"resourcetype"`
	Etag         string          `xml:"getetag"`
}

type davResourceType struct {
	Collection *struct{} `xml:"collection"`
}

type propfindEntry struct {
	href         string
	etag         string
	isCollection bool
}

func (c WebDAVClient) propfindURL(ctx context.Context, profile config.SyncProfile, dirURL string, depth int) ([]propfindEntry, error) {
	body := `<?xml version="1.0" encoding="UTF-8"?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:resourcetype/>
    <d:getetag/>
  </d:prop>
</d:propfind>`

	req, err := http.NewRequestWithContext(ctx, "PROPFIND", dirURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", fmt.Sprintf("%d", depth))
	if err := applyAuth(req, profile); err != nil {
		return nil, err
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != 207 {
		return nil, httpStatusError("PROPFIND", dirURL, resp)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ms davMultistatus
	if err := xml.Unmarshal(respBody, &ms); err != nil {
		return nil, fmt.Errorf("parse PROPFIND response: %w", err)
	}

	var entries []propfindEntry
	for _, r := range ms.Responses {
		isColl := r.Propstat.Prop.ResourceType.Collection != nil
		href := r.Href
		if href == "" {
			continue
		}
		entries = append(entries, propfindEntry{
			href:         href,
			etag:         strings.Trim(r.Propstat.Prop.Etag, `"`),
			isCollection: isColl,
		})
	}
	return entries, nil
}

func containsCollection(entries []propfindEntry, targetPath string) bool {
	targetPath = strings.TrimRight(targetPath, "/")
	for _, entry := range entries {
		entryPath := strings.TrimRight(urlPath(entry.href), "/")
		if entry.isCollection && entryPath == targetPath {
			return true
		}
		if strings.HasPrefix(entryPath, targetPath+"/") {
			return true
		}
	}
	return false
}

func urlPath(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err == nil && parsed.Path != "" {
		return parsed.Path
	}
	return strings.TrimSpace(raw)
}

func (c WebDAVClient) getFile(ctx context.Context, profile config.SyncProfile, fileURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", err
	}
	if err := applyAuth(req, profile); err != nil {
		return nil, "", err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, "", &RPCError{Code: ErrCodeNotFound, Message: "not found: " + fileURL}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", httpStatusError("GET", fileURL, resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	etag := resp.Header.Get("ETag")
	return body, etag, nil
}

func (c WebDAVClient) putFile(ctx context.Context, profile config.SyncProfile, fileURL string, data []byte, ifMatch string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fileURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	if err := applyAuth(req, profile); err != nil {
		return "", err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusPreconditionFailed {
		return "", &RPCError{Code: ErrCodeConflict, Message: "etag mismatch on PUT"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", httpStatusError("PUT", fileURL, resp)
	}
	return resp.Header.Get("ETag"), nil
}

func (c WebDAVClient) deleteFile(ctx context.Context, profile config.SyncProfile, fileURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fileURL, nil)
	if err != nil {
		return err
	}
	if err := applyAuth(req, profile); err != nil {
		return err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return httpStatusError("DELETE", fileURL, resp)
	}
	return nil
}

func (c WebDAVClient) moveFile(ctx context.Context, profile config.SyncProfile, srcURL, dstURL string) error {
	req, err := http.NewRequestWithContext(ctx, "MOVE", srcURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Destination", dstURL)
	req.Header.Set("Overwrite", "T")
	if err := applyAuth(req, profile); err != nil {
		return err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return httpStatusError("MOVE", srcURL+" -> "+dstURL, resp)
	}
	return nil
}

func (c WebDAVClient) mkcolParents(ctx context.Context, profile config.SyncProfile, baseURL, fileURL string) {
	parsedFile, err := url.Parse(fileURL)
	if err != nil {
		return
	}
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	dir := path.Dir(parsedFile.Path)
	base := parsedFile.Scheme + "://" + parsedFile.Host

	start := strings.TrimRight(parsedBase.Path, "/")
	if start == "" {
		start = "/"
	}

	current := start
	if current != "" && c.shouldEnsureDir(base+current) {
		if !c.mkcol(ctx, profile, base+current+"/") {
			return
		}
	}

	relDir := strings.TrimPrefix(dir, start)
	if relDir == dir {
		relDir = strings.Trim(dir, "/")
		current = ""
	} else {
		relDir = strings.Trim(relDir, "/")
	}
	if relDir == "" {
		return
	}
	segments := strings.Split(relDir, "/")
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		current += "/" + seg
		dirURL := base + current + "/"
		if !c.shouldEnsureDir(strings.TrimRight(dirURL, "/")) {
			continue
		}
		if !c.mkcol(ctx, profile, dirURL) {
			return
		}
	}
}

func (c WebDAVClient) shouldEnsureDir(key string) bool {
	if c.dirCache == nil {
		return true
	}
	c.dirCache.mu.Lock()
	defer c.dirCache.mu.Unlock()
	if _, exists := c.dirCache.seen[key]; exists {
		return false
	}
	c.dirCache.seen[key] = struct{}{}
	return true
}

func (c WebDAVClient) mkcol(ctx context.Context, profile config.SyncProfile, dirURL string) bool {
	mkReq, err := http.NewRequestWithContext(ctx, "MKCOL", dirURL, nil)
	if err != nil {
		return false
	}
	if err := applyAuth(mkReq, profile); err != nil {
		return false
	}
	resp, err := c.do(ctx, mkReq)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return true
}

// do issues req through the configured HTTP client with bounded retries on
// transient failures. It exists so that a single VPN hiccup, a server's
// momentary 503, or a Nextcloud rate-limit burst does not fail an entire
// sync run: most of the time the second or third attempt succeeds.
//
// Retries fire for:
//
//   - network errors that are not context cancellation (connection reset,
//     EOF mid-read, stalled read past ResponseHeaderTimeout);
//   - status codes 408, 425, 429, 500, 502, 503, 504 (the usual "try again"
//     class). 4xx client errors are returned immediately so the caller can
//     handle them (for example 412 Precondition Failed on If-Match).
//
// Request bodies are rewound via req.GetBody, which http.NewRequest sets
// automatically when the body is *bytes.Reader or *strings.Reader. If a body
// is present but not rewindable, we give up after the first attempt.
func (c WebDAVClient) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	const maxAttempts = 3
	backoff := [...]time.Duration{
		200 * time.Millisecond,
		600 * time.Millisecond,
		1500 * time.Millisecond,
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff[attempt-1]):
			}
			if req.Body != nil {
				if req.GetBody == nil {
					return nil, lastErr
				}
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = body
			}
		}

		resp, err := c.httpClient().Do(req)
		if err != nil {
			if !isRetriableNetError(err) {
				return nil, err
			}
			lastErr = err
			continue
		}
		if !isRetriableStatus(resp.StatusCode) {
			return resp, nil
		}
		_ = resp.Body.Close()
		lastErr = fmt.Errorf("transient status %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("%s %s: retries exhausted: %w", req.Method, req.URL.Redacted(), lastErr)
}

func isRetriableNetError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func isRetriableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusTooEarly,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func applyAuth(req *http.Request, profile config.SyncProfile) error {
	auth := strings.ToLower(strings.TrimSpace(profile.Auth))
	if auth == "" {
		auth = config.SyncAuthBasic
	}
	switch auth {
	case config.SyncAuthBasic:
		usernameEnv := strings.TrimSpace(profile.UsernameEnv)
		passwordEnv := strings.TrimSpace(profile.PasswordEnv)
		user, err := resolveCredentialValue(usernameEnv)
		if err != nil {
			return fmt.Errorf("webdav basic auth username env %s fallback failed: %w", usernameEnv, err)
		}
		pass, err := resolveCredentialValue(passwordEnv)
		if err != nil {
			return fmt.Errorf("webdav basic auth password env %s fallback failed: %w", passwordEnv, err)
		}
		if user == "" {
			return fmt.Errorf("webdav basic auth username env %s is not set", usernameEnv)
		}
		if pass == "" {
			return fmt.Errorf("webdav basic auth password env %s is not set", passwordEnv)
		}
		req.SetBasicAuth(user, pass)
	case config.SyncAuthBearer:
		tokenEnv := strings.TrimSpace(profile.TokenEnv)
		token, err := resolveCredentialValue(tokenEnv)
		if err != nil {
			return fmt.Errorf("webdav bearer auth token env %s fallback failed: %w", tokenEnv, err)
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("webdav bearer auth token env %s is not set", tokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func webdavBaseURL(profile config.SyncProfile, remoteRoot string) string {
	base := strings.TrimRight(strings.TrimSpace(profile.WebDAVURL), "/")
	root := strings.TrimSpace(profile.RemoteRoot)
	if root == "" {
		root = strings.TrimSpace(remoteRoot)
	}
	if root == "" {
		root = "/noteui"
	}
	root = strings.Trim(root, "/")
	if root == "" {
		return base
	}
	return base + "/" + root
}

func serverBaseURL(profile config.SyncProfile) string {
	u := strings.TrimSpace(profile.WebDAVURL)
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	return parsed.Scheme + "://" + parsed.Host
}

func resolveHref(serverBase, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return strings.TrimRight(serverBase, "/") + href
}

func escapePath(relPath string) string {
	segments := strings.Split(relPath, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

func contentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// httpStatusError builds an error for a failed WebDAV response that includes
// a bounded snippet of the response body. Nextcloud and most WebDAV servers
// return useful diagnostic text there (e.g. "Strict cookie not set",
// permission messages, XML multistatus errors), so surfacing it makes
// failures self-explanatory instead of "status 403".
func httpStatusError(method, target string, resp *http.Response) error {
	const maxBody = 4 * 1024
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	trimmed := strings.Join(strings.Fields(string(snippet)), " ")
	if trimmed == "" {
		return fmt.Errorf("%s %s: status %d", method, target, resp.StatusCode)
	}
	return fmt.Errorf("%s %s: status %d: %s", method, target, resp.StatusCode, trimmed)
}
