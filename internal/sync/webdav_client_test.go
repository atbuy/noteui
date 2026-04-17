package sync

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
)

type memWebDAV struct {
	mu       sync.Mutex
	files    map[string]memFile
	requests []memRequest
}

type memFile struct {
	body []byte
	etag string
}

type memRequest struct {
	method        string
	path          string
	authorization string
}

func newMemWebDAV() *memWebDAV {
	return &memWebDAV{files: make(map[string]memFile)}
}

func (m *memWebDAV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := r.URL.Path
	m.requests = append(m.requests, memRequest{
		method:        r.Method,
		path:          path,
		authorization: r.Header.Get("Authorization"),
	})
	switch r.Method {
	case http.MethodGet:
		f, ok := m.files[path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if f.etag != "" {
			w.Header().Set("ETag", `"`+f.etag+`"`)
		}
		_, _ = w.Write(f.body)
	case http.MethodPut:
		ifMatch := r.Header.Get("If-Match")
		if ifMatch != "" {
			existing, ok := m.files[path]
			if ok && `"`+existing.etag+`"` != ifMatch {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
		}
		body, _ := io.ReadAll(r.Body)
		etag := contentHash(body)
		m.files[path] = memFile{body: body, etag: etag}
		w.Header().Set("ETag", `"`+etag+`"`)
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		if _, ok := m.files[path]; !ok {
			http.NotFound(w, r)
			return
		}
		delete(m.files, path)
		w.WriteHeader(http.StatusNoContent)
	case "MKCOL":
		w.WriteHeader(http.StatusCreated)
	case "MOVE":
		dest := r.Header.Get("Destination")
		f, ok := m.files[path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		parsed := dest
		if strings.HasPrefix(dest, "http") {
			idx := strings.Index(dest[8:], "/")
			if idx >= 0 {
				parsed = dest[8+idx:]
			}
		}
		m.files[parsed] = f
		delete(m.files, path)
		w.WriteHeader(http.StatusCreated)
	case "PROPFIND":
		prefix := strings.TrimRight(path, "/")
		var buf strings.Builder
		buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		buf.WriteString(`<d:multistatus xmlns:d="DAV:">`)
		for p, f := range m.files {
			if !strings.HasPrefix(p, prefix+"/") && p != prefix {
				continue
			}
			buf.WriteString(`<d:response>`)
			buf.WriteString(`<d:href>` + p + `</d:href>`)
			buf.WriteString(`<d:propstat>`)
			buf.WriteString(`<d:prop>`)
			buf.WriteString(`<d:resourcetype/>`)
			buf.WriteString(`<d:getetag>"` + f.etag + `"</d:getetag>`)
			buf.WriteString(`</d:prop>`)
			buf.WriteString(`<d:status>HTTP/1.1 200 OK</d:status>`)
			buf.WriteString(`</d:propstat>`)
			buf.WriteString(`</d:response>`)
		}
		buf.WriteString(`</d:multistatus>`)
		w.WriteHeader(207)
		_, _ = w.Write([]byte(buf.String()))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func testWebDAVProfile(serverURL string) config.SyncProfile {
	return config.SyncProfile{
		Kind:      "webdav",
		WebDAVURL: serverURL,
		Auth:      "none",
	}
}

func TestWebDAVRegisterAndFetchNote(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}
	ctx := context.Background()

	regResp, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "work/plan.md",
		Content:    "---\ntitle: Plan\n---\nHello",
		Encrypted:  false,
	})
	require.NoError(t, err)
	require.NotEmpty(t, regResp.ID)
	require.NotEmpty(t, regResp.Revision)

	fetchResp, err := client.FetchNote(ctx, profile, FetchNoteRequest{
		RemoteRoot: "/noteui",
		NoteID:     regResp.ID,
	})
	require.NoError(t, err)
	require.Equal(t, regResp.ID, fetchResp.Note.ID)
	require.Equal(t, "work/plan.md", fetchResp.Note.RelPath)
	require.Contains(t, fetchResp.Note.Content, "Hello")
}

func TestWebDAVPullIndex(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}
	ctx := context.Background()

	_, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "notes/a.md",
		Content:    "---\ntitle: Alpha\n---\nContent A",
	})
	require.NoError(t, err)

	idx, err := client.PullIndex(ctx, profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.Len(t, idx.Notes, 1)
	require.Equal(t, "notes/a.md", idx.Notes[0].RelPath)
}

func TestWebDAVPullIndexEmptyRemoteDoesNotRequireMetadataDir(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}
	ctx := context.Background()

	idx, err := client.PullIndex(ctx, profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.Empty(t, idx.Notes)

	var propfindPaths []string
	for _, req := range store.requests {
		if req.method == "PROPFIND" {
			propfindPaths = append(propfindPaths, req.path)
		}
	}
	require.Equal(t, []string{"/noteui"}, propfindPaths)
}

func TestWebDAVPullIndexRequiresConfiguredEnvValues(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	t.Setenv("NOTEUI_NEXTCLOUD_USERNAME", "")
	t.Setenv("NOTEUI_NEXTCLOUD_PASSWORD", "")

	profile := config.SyncProfile{
		Kind:        config.SyncKindWebDAV,
		WebDAVURL:   srv.URL,
		Auth:        config.SyncAuthBasic,
		UsernameEnv: "NOTEUI_NEXTCLOUD_USERNAME",
		PasswordEnv: "NOTEUI_NEXTCLOUD_PASSWORD",
	}
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "webdav basic auth username env NOTEUI_NEXTCLOUD_USERNAME is not set")
}

func TestWebDAVPullIndexSendsBasicAuthFromEnv(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	t.Setenv("NOTEUI_NEXTCLOUD_USERNAME", "filip")
	t.Setenv("NOTEUI_NEXTCLOUD_PASSWORD", "app-password")

	profile := config.SyncProfile{
		Kind:        config.SyncKindWebDAV,
		WebDAVURL:   srv.URL,
		Auth:        config.SyncAuthBasic,
		UsernameEnv: "NOTEUI_NEXTCLOUD_USERNAME",
		PasswordEnv: "NOTEUI_NEXTCLOUD_PASSWORD",
	}
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.NotEmpty(t, store.requests)
	require.NotEmpty(t, store.requests[0].authorization)
	require.True(t, strings.HasPrefix(store.requests[0].authorization, "Basic "))
}

func TestWebDAVPullIndexReadsBasicAuthFromSecretsFileWhenEnvUnset(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	t.Setenv("NOTEUI_NEXTCLOUD_USERNAME", "")
	t.Setenv("NOTEUI_NEXTCLOUD_PASSWORD", "")
	writeSecretsFile(t, strings.Join([]string{
		`NOTEUI_NEXTCLOUD_USERNAME = "filip"`,
		`NOTEUI_NEXTCLOUD_PASSWORD = "app-password"`,
	}, "\n"))

	profile := config.SyncProfile{
		Kind:        config.SyncKindWebDAV,
		WebDAVURL:   srv.URL,
		Auth:        config.SyncAuthBasic,
		UsernameEnv: "NOTEUI_NEXTCLOUD_USERNAME",
		PasswordEnv: "NOTEUI_NEXTCLOUD_PASSWORD",
	}
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.NotEmpty(t, store.requests)

	scheme, encoded, found := strings.Cut(store.requests[0].authorization, " ")
	require.True(t, found)
	require.Equal(t, "Basic", scheme)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	require.Equal(t, "filip:app-password", string(decoded))
}

func TestWebDAVPullIndexSendsBearerTokenFromEnv(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	t.Setenv("NOTEUI_WEBDAV_TOKEN", "app-token-xyz")

	profile := config.SyncProfile{
		Kind:      config.SyncKindWebDAV,
		WebDAVURL: srv.URL,
		Auth:      config.SyncAuthBearer,
		TokenEnv:  "NOTEUI_WEBDAV_TOKEN",
	}
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.NotEmpty(t, store.requests)
	require.Equal(t, "Bearer app-token-xyz", store.requests[0].authorization)
}

func TestWebDAVPullIndexReadsBearerTokenFromSecretsFileWhenEnvUnset(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	t.Setenv("NOTEUI_WEBDAV_TOKEN", "")
	writeSecretsFile(t, `NOTEUI_WEBDAV_TOKEN = "app-token-from-file"`)

	profile := config.SyncProfile{
		Kind:      config.SyncKindWebDAV,
		WebDAVURL: srv.URL,
		Auth:      config.SyncAuthBearer,
		TokenEnv:  "NOTEUI_WEBDAV_TOKEN",
	}
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.NotEmpty(t, store.requests)
	require.Equal(t, "Bearer app-token-from-file", store.requests[0].authorization)
}

func TestWebDAVPullIndexBearerAuthRequiresConfiguredTokenEnv(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	t.Setenv("NOTEUI_WEBDAV_TOKEN", "")

	profile := config.SyncProfile{
		Kind:      config.SyncKindWebDAV,
		WebDAVURL: srv.URL,
		Auth:      config.SyncAuthBearer,
		TokenEnv:  "NOTEUI_WEBDAV_TOKEN",
	}
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "webdav bearer auth token env NOTEUI_WEBDAV_TOKEN is not set")
}

func TestWebDAVPullIndexReportsSecretsFileParseError(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	t.Setenv("NOTEUI_WEBDAV_TOKEN", "")
	writeSecretsFile(t, `NOTEUI_WEBDAV_TOKEN = [`)

	profile := config.SyncProfile{
		Kind:      config.SyncKindWebDAV,
		WebDAVURL: srv.URL,
		Auth:      config.SyncAuthBearer,
		TokenEnv:  "NOTEUI_WEBDAV_TOKEN",
	}
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "fallback failed")
	require.Contains(t, err.Error(), "secrets.toml")
}

func TestWebDAVPushNote(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()
	ctx := context.Background()

	reg, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "note.md",
		Content:    "v1",
	})
	require.NoError(t, err)

	pushResp, err := client.PushNote(ctx, profile, PushNoteRequest{
		RemoteRoot:       "/noteui",
		NoteID:           reg.ID,
		ExpectedRevision: reg.Revision,
		RelPath:          "note.md",
		Content:          "v2",
	})
	require.NoError(t, err)
	require.NotEqual(t, reg.Revision, pushResp.Revision)

	fetched, err := client.FetchNote(ctx, profile, FetchNoteRequest{
		RemoteRoot: "/noteui",
		NoteID:     reg.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "v2", fetched.Note.Content)
}

func writeSecretsFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := filepath.Join(dir, "noteui", "secrets.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestWebDAVPushNoteRevisionMismatch(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()
	ctx := context.Background()

	reg, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "note.md",
		Content:    "v1",
	})
	require.NoError(t, err)

	_, err = client.PushNote(ctx, profile, PushNoteRequest{
		RemoteRoot:       "/noteui",
		NoteID:           reg.ID,
		ExpectedRevision: "bogus-rev",
		RelPath:          "note.md",
		Content:          "v2",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "revision mismatch")
}

func TestWebDAVDeleteNote(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()
	ctx := context.Background()

	reg, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "delete-me.md",
		Content:    "bye",
	})
	require.NoError(t, err)

	_, err = client.DeleteNote(ctx, profile, DeleteNoteRequest{
		RemoteRoot: "/noteui",
		NoteID:     reg.ID,
	})
	require.NoError(t, err)

	_, err = client.FetchNote(ctx, profile, FetchNoteRequest{
		RemoteRoot: "/noteui",
		NoteID:     reg.ID,
	})
	require.Error(t, err)
}

func TestWebDAVUpdateNotePath(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()
	ctx := context.Background()

	reg, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "old/path.md",
		Content:    "content",
	})
	require.NoError(t, err)

	upResp, err := client.UpdateNotePath(ctx, profile, UpdateNotePathRequest{
		RemoteRoot:       "/noteui",
		NoteID:           reg.ID,
		ExpectedRevision: reg.Revision,
		RelPath:          "new/path.md",
	})
	require.NoError(t, err)
	require.NotEmpty(t, upResp.Revision)

	fetched, err := client.FetchNote(ctx, profile, FetchNoteRequest{
		RemoteRoot: "/noteui",
		NoteID:     reg.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "new/path.md", fetched.Note.RelPath)
	require.Equal(t, "content", fetched.Note.Content)
}

func TestWebDAVPinsGetPut(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()
	ctx := context.Background()

	pins := Pins{PinnedNoteIDs: []string{"n_abc"}, PinnedCategories: []string{"work"}}
	putResp, err := client.PinsPut(ctx, profile, PinsPutRequest{RemoteRoot: "/noteui", Pins: pins})
	require.NoError(t, err)
	require.Equal(t, pins, putResp.Pins)

	getResp, err := client.PinsGet(ctx, profile, PinsGetRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.Equal(t, []string{"n_abc"}, getResp.Pins.PinnedNoteIDs)
	require.Equal(t, []string{"work"}, getResp.Pins.PinnedCategories)
}

func TestWebDAVExternalEditDetected(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()
	ctx := context.Background()

	reg, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "ext.md",
		Content:    "original",
	})
	require.NoError(t, err)

	// Simulate external edit via Nextcloud (direct file write)
	noteKey := "/noteui/ext.md"
	staleEtag := store.files[noteKey].etag
	newContent := []byte("edited externally")
	store.files[noteKey] = memFile{body: newContent, etag: staleEtag}

	fetched, err := client.FetchNote(ctx, profile, FetchNoteRequest{
		RemoteRoot: "/noteui",
		NoteID:     reg.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "edited externally", fetched.Note.Content)
	require.NotEqual(t, reg.Revision, fetched.Note.Revision)
}

func TestWebDAVContentHashFallback(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client()}
	client.dirCache = newWebDAVDirCache()
	ctx := context.Background()

	reg, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "hash.md",
		Content:    "test content",
	})
	require.NoError(t, err)

	// Strip ETag to simulate server without ETag support
	for k, f := range store.files {
		f.etag = ""
		store.files[k] = f
	}

	fetched, err := client.FetchNote(ctx, profile, FetchNoteRequest{
		RemoteRoot: "/noteui",
		NoteID:     reg.ID,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(fetched.Note.Revision, "sha256:"))
}

func TestWebDAVRegisterNoteAvoidsMKCOLAboveConfiguredBase(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := config.SyncProfile{
		Kind:      config.SyncKindWebDAV,
		WebDAVURL: srv.URL + "/remote.php/dav/files/alice",
		Auth:      config.SyncAuthNone,
	}
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}
	ctx := context.Background()

	_, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/Notes",
		RelPath:    "work/plan.md",
		Content:    "hello",
	})
	require.NoError(t, err)

	var mkcolPaths []string
	for _, req := range store.requests {
		if req.method == "MKCOL" {
			mkcolPaths = append(mkcolPaths, req.path)
		}
	}
	require.Equal(t, []string{
		"/remote.php/dav/files/alice/Notes/",
		"/remote.php/dav/files/alice/Notes/work/",
		"/remote.php/dav/files/alice/Notes/.noteui-sync/",
		"/remote.php/dav/files/alice/Notes/.noteui-sync/notes/",
	}, mkcolPaths)
}

func TestNewClientReturnsCorrectType(t *testing.T) {
	sshProfile := config.SyncProfile{Kind: "ssh"}
	davProfile := config.SyncProfile{Kind: "webdav"}
	emptyProfile := config.SyncProfile{}

	_, isSSH := NewClient(sshProfile).(SSHClient)
	require.True(t, isSSH)

	_, isDAV := NewClient(davProfile).(WebDAVClient)
	require.True(t, isDAV)

	_, isSSHDefault := NewClient(emptyProfile).(SSHClient)
	require.True(t, isSSHDefault)
}

func TestWebDAVNoteMapping(t *testing.T) {
	mapping := webdavNoteMapping{ID: "n_abc", RelPath: "notes/test.md", Encrypted: true}
	data, err := json.Marshal(mapping)
	require.NoError(t, err)

	var decoded webdavNoteMapping
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, mapping, decoded)
}

func TestEscapePath(t *testing.T) {
	require.Equal(t, "notes/my%20file.md", escapePath("notes/my file.md"))
	require.Equal(t, "simple.md", escapePath("simple.md"))
	require.Equal(t, "a/b/c.md", escapePath("a/b/c.md"))
}

// Nextcloud's session middleware sets nc_session_id on the first hit and
// rejects follow-ups that do not carry it. NewClient must give WebDAVClient
// an http.Client with a cookie jar so the cookie survives across requests
// and across a 307 redirect inside a single call. Without the jar the
// second request would be treated as a fresh session and loop on redirects.
func TestNewClientWebDAVPersistsSessionCookieAcrossRequests(t *testing.T) {
	var (
		mu           sync.Mutex
		cookieless   int
		withCookie   int
		redirectsHit int
	)
	store := newMemWebDAV()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("nc_session_id")
		mu.Lock()
		if cookie == nil {
			cookieless++
		} else {
			withCookie++
		}
		mu.Unlock()

		if cookie == nil {
			mu.Lock()
			redirectsHit++
			mu.Unlock()
			http.SetCookie(w, &http.Cookie{Name: "nc_session_id", Value: "session-value", Path: "/"})
			http.Redirect(w, r, r.URL.Path, http.StatusTemporaryRedirect)
			return
		}
		store.ServeHTTP(w, r)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	profile := config.SyncProfile{
		Kind:      config.SyncKindWebDAV,
		WebDAVURL: srv.URL,
		Auth:      "none",
	}
	client := NewClient(profile)
	ctx := context.Background()

	_, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui",
		RelPath:    "notes/a.md",
		Content:    "hello",
	})
	require.NoError(t, err)

	idx, err := client.PullIndex(ctx, profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.Len(t, idx.Notes, 1)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, cookieless, "only the very first request should lack the cookie")
	require.Equal(t, 1, redirectsHit, "jar should avoid further redirect loops after first hit")
	require.Greater(t, withCookie, 1, "jar should replay cookie on every follow-up request")
}

// newWebDAVHTTPClient must configure an overall request timeout and a cookie
// jar. Without the timeout, sync hangs forever on a stalled VPN connection;
// without the jar, Nextcloud treats every request as a fresh session. Verify
// the wiring structurally so future refactors cannot silently drop either.
func TestNewClientWebDAVHasTimeoutAndCookieJar(t *testing.T) {
	profile := config.SyncProfile{
		Kind:      config.SyncKindWebDAV,
		WebDAVURL: "https://example.invalid/dav",
		Auth:      "none",
	}
	client := NewClient(profile)

	wc, ok := client.(WebDAVClient)
	require.True(t, ok)
	require.NotNil(t, wc.HTTP)
	require.Greater(t, wc.HTTP.Timeout, time.Duration(0), "overall request timeout must be set")
	require.NotNil(t, wc.HTTP.Jar, "cookie jar must be set")
	require.NotNil(t, wc.HTTP.Transport, "transport must be configured for dial / TLS timeouts")
}

// When a WebDAV request fails with a non-2xx status, the returned error must
// include the server's response body (e.g. Nextcloud XML diagnostics) so the
// caller can see *why* the request was rejected, not merely the status code.
func TestWebDAVErrorIncludesResponseBodySnippet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(
			"<?xml version=\"1.0\"?>\n" +
				"<d:error xmlns:d=\"DAV:\">\n" +
				"  <d:message>Strict cookie nc_session_id not found</d:message>\n" +
				"</d:error>\n"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.Error(t, err)
	msg := err.Error()
	require.Contains(t, msg, "status 403")
	require.Contains(t, msg, "Strict cookie nc_session_id not found")
}

// A single burst of 503s (common when Nextcloud is momentarily overloaded or
// a VPN interrupts a TCP connection) must not fail the whole sync: the
// retry helper should swallow a couple of transient failures and succeed on
// the next attempt.
func TestWebDAVRetriesTransientServerErrors(t *testing.T) {
	store := newMemWebDAV()
	var mu sync.Mutex
	var propfindCalls int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PROPFIND" {
			mu.Lock()
			propfindCalls++
			n := propfindCalls
			mu.Unlock()
			if n <= 2 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		store.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, propfindCalls, 3, "expected two failed PROPFINDs plus one success")
}

// 4xx responses (here 403) are a definitive answer from the server, not a
// transient blip: retrying would only hammer the server. Verify the retry
// helper hands them back immediately.
func TestWebDAVDoesNotRetryNonTransientStatus(t *testing.T) {
	var mu sync.Mutex
	var calls int
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("denied"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}

	_, err := client.PullIndex(context.Background(), profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.Error(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, calls, "403 must not trigger retry")
}

// When the remote contains a corrupt mapping file (truncated write, manual
// edit, etc), PullIndex must still return the notes it *could* load and
// report the skipped count so the sync engine can surface a warning instead
// of silently hiding the broken entry.
func TestWebDAVPullIndexSkippedCountReportsBadMappings(t *testing.T) {
	store := newMemWebDAV()
	srv := httptest.NewServer(store)
	defer srv.Close()

	profile := testWebDAVProfile(srv.URL)
	client := WebDAVClient{HTTP: srv.Client(), dirCache: newWebDAVDirCache()}
	ctx := context.Background()

	_, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui", RelPath: "notes/a.md", Content: "A",
	})
	require.NoError(t, err)
	_, err = client.RegisterNote(ctx, profile, RegisterNoteRequest{
		RemoteRoot: "/noteui", RelPath: "notes/b.md", Content: "B",
	})
	require.NoError(t, err)

	store.files["/noteui/.noteui-sync/notes/corrupt.json"] = memFile{
		body: []byte("{not json"),
		etag: "corrupt",
	}

	idx, err := client.PullIndex(ctx, profile, PullIndexRequest{RemoteRoot: "/noteui"})
	require.NoError(t, err)
	require.Len(t, idx.Notes, 2)
	require.Equal(t, 1, idx.SkippedCount)
	require.Equal(t, "notes/a.md", idx.Notes[0].RelPath)
	require.Equal(t, "notes/b.md", idx.Notes[1].RelPath)
}

func TestWebDAVBaseURL(t *testing.T) {
	p := config.SyncProfile{WebDAVURL: "https://cloud.example.com/dav"}
	require.Equal(t, "https://cloud.example.com/dav/noteui", webdavBaseURL(p, ""))
	require.Equal(t, "https://cloud.example.com/dav/custom", webdavBaseURL(p, "/custom"))

	p2 := config.SyncProfile{WebDAVURL: "https://cloud.example.com/dav", RemoteRoot: "/my/notes"}
	require.Equal(t, "https://cloud.example.com/dav/my/notes", webdavBaseURL(p2, ""))
	require.Equal(t, "https://cloud.example.com/dav/my/notes", webdavBaseURL(p2, "/home/user/.local"))
}
