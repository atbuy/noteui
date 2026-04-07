package notes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExtractTodoItemsParsesMetadataAndLineOffsets(t *testing.T) {
	raw := `---
tags: work
---
# Todo

- [ ] Ship release [p1] [due:2026-04-12]
- [x] Closed item [p3]
- [ ] Inbox cleanup
`
	items := ExtractTodoItems(raw, true)
	require.Len(t, items, 2)
	require.Equal(t, 5, items[0].Line)
	require.Equal(t, "Ship release", items[0].DisplayText)
	require.Equal(t, 1, items[0].Metadata.Priority)
	require.Equal(t, "2026-04-12", items[0].Metadata.DueDate)
	require.Equal(t, time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), items[0].Metadata.DueTime)
	require.Equal(t, 7, items[1].Line)
	require.Equal(t, "Inbox cleanup", items[1].DisplayText)
}

func TestParseTodoMetadataAcceptsArbitraryPriority(t *testing.T) {
	display, metadata := ParseTodoMetadata("Later task [p5]")
	require.Equal(t, "Later task", display)
	require.Equal(t, 5, metadata.Priority)
}

func TestParseTodoMetadataLeavesMalformedDueDateInText(t *testing.T) {
	display, metadata := ParseTodoMetadata("Review docs [due:soon] [p2]")
	require.Equal(t, "Review docs [due:soon]", display)
	require.Equal(t, 2, metadata.Priority)
	require.Empty(t, metadata.DueDate)
	require.True(t, metadata.DueTime.IsZero())
}
