package notes

import "atbuy/noteui/internal/notes/todo"

type (
	TodoMetadata = todo.Metadata
	TodoItem     = todo.Item
)

func ExtractTodoItems(raw string, openOnly bool) []TodoItem { return todo.ExtractItems(raw, openOnly) }
func ParseTodoMetadata(text string) (string, TodoMetadata)  { return todo.ParseMetadata(text) }
