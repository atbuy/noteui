package sync

import (
	"net/url"
	"strings"
)

const webDAVRevisionPrefix = "davrev:"

type parsedRevision struct {
	raw  string
	etag string
	hash string
}

func buildWebDAVRevision(etag string, body []byte) string {
	hash := contentHash(body)
	etag = strings.Trim(strings.TrimSpace(etag), `"`)
	if etag == "" {
		return hash
	}
	values := url.Values{}
	values.Set("etag", etag)
	values.Set("hash", hash)
	return webDAVRevisionPrefix + values.Encode()
}

func sameRevision(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == right {
		return true
	}

	a := parseRevision(left)
	b := parseRevision(right)

	switch {
	case a.hash != "" && b.hash != "":
		return a.hash == b.hash
	case a.hash != "" && b.hash == "" && b.etag != "":
		return a.etag != "" && a.etag == b.etag
	case b.hash != "" && a.hash == "" && a.etag != "":
		return b.etag != "" && b.etag == a.etag
	case a.etag != "" && b.etag != "":
		return a.etag == b.etag
	default:
		return a.raw == b.raw
	}
}

func remoteContentChanged(rec NoteRecord, remoteRevision string) bool {
	remote := parseRevision(remoteRevision)
	lastSyncedHash := strings.TrimSpace(rec.LastSyncedHash)
	if remote.hash != "" && lastSyncedHash != "" {
		return remote.hash != lastSyncedHash
	}
	return !sameRevision(rec.RemoteRev, remoteRevision)
}

func parseRevision(raw string) parsedRevision {
	raw = strings.TrimSpace(raw)
	out := parsedRevision{raw: raw}
	switch {
	case raw == "":
		return out
	case strings.HasPrefix(raw, webDAVRevisionPrefix):
		values, err := url.ParseQuery(strings.TrimPrefix(raw, webDAVRevisionPrefix))
		if err != nil {
			out.etag = raw
			return out
		}
		out.etag = strings.TrimSpace(values.Get("etag"))
		out.hash = strings.TrimSpace(values.Get("hash"))
		if out.etag == "" && out.hash == "" {
			out.etag = raw
		}
	case strings.HasPrefix(raw, "sha256:"):
		out.hash = raw
	default:
		out.etag = raw
	}
	return out
}
