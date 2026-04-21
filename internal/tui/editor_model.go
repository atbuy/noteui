package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
)

const editorUndoLimit = 200

type editorMode int

const (
	editorModeNormal editorMode = iota
	editorModeInsert
	editorModeVisualChar
	editorModeVisualLine
	editorModeCmdline
	editorModeSearch
)

type editorPoint struct {
	row int
	col int
}

type editorUndoSnapshot struct {
	lines     [][]rune
	row       int
	col       int
	preferCol int
	viewTop   int
	viewLeft  int
}

type editorRegister struct {
	text     string
	linewise bool
}

type editorLoadedMsg struct {
	path      string
	relPath   string
	content   string
	hash      string
	modTime   time.Time
	encrypted bool
	isTemp    bool
	err       error
}

type editorSavedMsg struct {
	oldPath    string
	newPath    string
	hash       string
	modTime    time.Time
	closeAfter bool
	discarded  bool
}

type (
	editorSaveErrMsg  struct{ err error }
	editorConflictMsg struct{}
)

type editorReloadedMsg struct {
	content string
	hash    string
	modTime time.Time
}

type (
	editorReloadErrMsg        struct{ err error }
	editorClosedMsg           struct{ discarded bool }
	editorLinkPickerMsg       struct{}
	editorURLPromptMsg        struct{}
	editorToggleFullscreenMsg struct{}
	editorRelativeNumbersMsg  struct{ enabled bool }
)

type EditorModel struct {
	lines         [][]rune
	row           int
	col           int
	preferCol     int
	mode          editorMode
	undoStack     []editorUndoSnapshot
	redoStack     []editorUndoSnapshot
	visualAnchor  editorPoint
	cmdBuf        []rune
	searchBuf     []rune
	lastSearch    []rune
	lastSearchFwd bool
	viewTop       int
	viewLeft      int
	width         int
	height        int
	contentHeight int
	path          string
	relPath       string
	rootDir       string
	encrypted     bool
	isTemp        bool
	passphrase    string
	loadHash      string
	loadModTime   time.Time
	dirty         bool
	pendingPrefix rune
	pendingOp     rune
	pendingCount  int
	pendingT      bool
	registers     editorRegister
	status        string

	// Rendered-mode fields (populated when renderMode is true).
	frontmatter         string       // raw "---...---" block preserved from file
	renderMode          bool         // show rendered markdown in normal mode
	renderedDoc         *RenderedDoc // derived from e.lines; nil until first render
	renderViewTop       int          // scroll offset for the rendered view
	lineNumbersEnabled  bool         // mirrors previewLineNumbersEnabled
	relativeLineNumbers bool         // show distances from cursor instead of absolute numbers
}

func NewEditorModel(path, relPath, rootDir, content string, width, height int, encrypted bool, passphrase string, isTemp bool) EditorModel {
	e := EditorModel{
		path:          path,
		relPath:       relPath,
		rootDir:       rootDir,
		encrypted:     encrypted,
		passphrase:    passphrase,
		isTemp:        isTemp,
		mode:          editorModeNormal,
		lastSearchFwd: true,
		status:        "editing in app",
		renderMode:    true,
	}

	// Strip frontmatter so the rendered view only sees body content.
	// The frontmatter is preserved and prepended on Content().
	fm, body := notes.SplitFrontMatter(content)
	e.frontmatter = fm
	e.setFromRunes(editorNormalizeContent(body))

	e.Resize(width, height)
	e.row = 0
	e.col = 0
	e.preferCol = 0
	e.recomputeRenderedDoc()
	if e.renderedDoc == nil || len(e.renderedDoc.Blocks) == 0 {
		// No renderable content: fall back to raw mode.
		e.renderMode = false
	}
	return e
}

func (e *EditorModel) Resize(width, height int) {
	e.width = max(20, width)
	e.height = max(3, height)
	e.contentHeight = max(1, e.height-2)
	if e.renderMode {
		e.recomputeRenderedDoc()
		e.syncRenderViewTop()
	}
	e.ensureVisible()
}

func (e EditorModel) Content() string {
	body := editorJoinLines(e.lines)
	if e.frontmatter != "" {
		return e.frontmatter + "\n" + body
	}
	return body
}

func (e *EditorModel) markLoaded(hash string, modTime time.Time) {
	e.loadHash = hash
	e.loadModTime = modTime
	e.dirty = false
}

func (e *EditorModel) markSaved(newPath, hash string, modTime time.Time) {
	e.path = newPath
	e.relPath = editorRelativePath(e.rootDir, newPath, e.isTemp)
	e.loadHash = hash
	e.loadModTime = modTime
	e.dirty = false
	e.mode = editorModeNormal
	e.pendingPrefix = 0
	e.pendingOp = 0
	e.pendingCount = 0
	e.cmdBuf = nil
	e.searchBuf = nil
}

func (e *EditorModel) applyReload(content, hash string, modTime time.Time) {
	fm, body := notes.SplitFrontMatter(content)
	e.frontmatter = fm
	e.setFromRunes(editorNormalizeContent(body))
	e.row = 0
	e.col = 0
	e.preferCol = 0
	e.viewTop = 0
	e.viewLeft = 0
	e.renderViewTop = 0
	e.undoStack = nil
	e.redoStack = nil
	e.mode = editorModeNormal
	e.pendingPrefix = 0
	e.pendingOp = 0
	e.pendingCount = 0
	e.cmdBuf = nil
	e.searchBuf = nil
	e.markLoaded(hash, modTime)
	if e.renderMode {
		e.recomputeRenderedDoc()
		if e.renderedDoc == nil || len(e.renderedDoc.Blocks) == 0 {
			e.renderMode = false
		}
	}
	e.status = "reloaded"
}

func (e *EditorModel) setStatus(text string) {
	e.status = text
}

func (e EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch e.mode {
		case editorModeInsert:
			return e.updateInsert(msg)
		case editorModeCmdline:
			return e.updateCmdline(msg)
		case editorModeSearch:
			return e.updateSearch(msg)
		case editorModeVisualChar, editorModeVisualLine:
			return e.updateVisual(msg)
		default:
			return e.updateNormal(msg)
		}
	case tea.WindowSizeMsg:
		e.Resize(msg.Width, msg.Height)
	}
	return e, nil
}

func (e EditorModel) View() string {
	if e.renderMode && e.renderedDoc != nil {
		return e.viewRendered()
	}
	gw := 0
	var gutterStyle lipgloss.Style
	var digits int
	if e.lineNumbersEnabled && len(e.lines) > 0 {
		digits = len(fmt.Sprintf("%d", max(1, len(e.lines))))
		gw = digits + 1
		gutterStyle = lipgloss.NewStyle().Foreground(mutedColor).Background(bgSoftColor)
	}
	contentWidth := e.width - gw
	body := make([]string, 0, e.contentHeight+2)
	for i := 0; i < e.contentHeight; i++ {
		lineIdx := e.viewTop + i
		line := e.renderLine(lineIdx, contentWidth)
		if gw > 0 {
			line = gutterStyle.Render(fmt.Sprintf("%*d ", digits, e.gutterLineNumber(lineIdx))) + line
		}
		body = append(body, line)
	}
	body = append(body, e.renderCommandLine())
	body = append(body, e.renderStatusLine())
	return lipgloss.NewStyle().
		Width(e.width).
		Height(e.height).
		Background(bgSoftColor).
		Render(strings.Join(body, "\n"))
}

func (e EditorModel) modeLabel() string {
	switch e.mode {
	case editorModeInsert:
		return "INSERT"
	case editorModeVisualChar:
		return "VISUAL"
	case editorModeVisualLine:
		return "V-LINE"
	case editorModeCmdline:
		return "CMD"
	case editorModeSearch:
		if e.lastSearchFwd {
			return "SEARCH"
		}
		return "SEARCH ?"
	default:
		return "NORMAL"
	}
}

func (e EditorModel) displayPath() string {
	if strings.TrimSpace(e.relPath) == "" {
		if strings.TrimSpace(e.path) == "" {
			return "<buffer>"
		}
		return filepath.Base(e.path)
	}
	if e.isTemp {
		return filepath.ToSlash(filepath.Join(".tmp", e.relPath))
	}
	return filepath.ToSlash(e.relPath)
}

func (e EditorModel) renderCommandLine() string {
	text := ""
	switch e.mode {
	case editorModeCmdline:
		text = ":" + string(e.cmdBuf)
	case editorModeSearch:
		prefix := "/"
		if !e.lastSearchFwd {
			prefix = "?"
		}
		text = prefix + string(e.searchBuf)
	}
	if text != "" {
		return editorFillLine(text, e.width, bgSoftColor, accentColor)
	}
	return editorFillLine(text, e.width, bgSoftColor, textColor)
}

func (e EditorModel) renderStatusLine() string {
	dirty := ""
	if e.dirty {
		dirty = " [+]"
	}
	status := strings.TrimSpace(e.status)
	text := fmt.Sprintf("-- %s -- %s%s  %d:%d", e.modeLabel(), e.displayPath(), dirty, e.row+1, e.cursorDisplayCol()+1)
	if status != "" {
		text += "  |  " + status
	}
	return editorFillLine(text, e.width, bgColor, mutedColor)
}

func (e EditorModel) renderLine(lineIdx, width int) string {
	bg := bgSoftColor
	baseStyle := lipgloss.NewStyle().Background(bg).Foreground(textColor)
	selectedStyle := lipgloss.NewStyle().Background(selectedBgColor).Foreground(selectedFgColor).Bold(boldSelected)
	cursorStyle := lipgloss.NewStyle().Background(accentSoftColor).Foreground(bgSoftColor).Bold(true)
	visualCursorStyle := lipgloss.NewStyle().Background(selectedFgColor).Foreground(selectedBgColor).Bold(true)

	if lineIdx < 0 || lineIdx >= len(e.lines) {
		return editorFillLine("", width, bg, textColor)
	}

	line := e.lines[lineIdx]
	var b strings.Builder
	for screenCol := 0; screenCol < width; screenCol++ {
		col := e.viewLeft + screenCol
		ch := ' '
		hasRune := col < len(line)
		if hasRune {
			ch = line[col]
		}
		text := string(ch)

		selected := e.isSelectedCell(lineIdx, col)
		isCursor := e.cursorOnCell(lineIdx, col, hasRune)

		style := baseStyle
		if selected {
			style = selectedStyle
		}
		if isCursor {
			if e.mode == editorModeVisualChar || e.mode == editorModeVisualLine {
				style = visualCursorStyle
			} else {
				style = cursorStyle
			}
		}

		b.WriteString(style.Render(text))
	}
	return b.String()
}

func (e EditorModel) cursorDisplayCol() int {
	if e.mode == editorModeInsert {
		return min(e.col, len(e.currentLine()))
	}
	if len(e.currentLine()) == 0 {
		return 0
	}
	return min(e.col, len(e.currentLine())-1)
}

func (e EditorModel) cursorOnCell(lineIdx, col int, hasRune bool) bool {
	if lineIdx != e.row {
		return false
	}
	switch e.mode {
	case editorModeInsert:
		return col == e.col
	default:
		if len(e.currentLine()) == 0 {
			return col == 0
		}
		if hasRune {
			return col == e.col
		}
		return col == e.col && e.col == len(e.currentLine())
	}
}

func (e EditorModel) isSelectedCell(lineIdx, col int) bool {
	switch e.mode {
	case editorModeVisualLine:
		start, end := e.selectedLineBounds()
		return lineIdx >= start && lineIdx <= end
	case editorModeVisualChar:
		start, end, ok := e.visualSelectionOffsets()
		if !ok {
			return false
		}
		if lineIdx >= len(e.lines) {
			return false
		}
		cellOffset := e.pointToOffset(editorPoint{row: lineIdx, col: min(col, len(e.lines[lineIdx]))})
		return cellOffset >= start && cellOffset < end
	default:
		return false
	}
}

func (e EditorModel) updateNormal(msg tea.KeyMsg) (EditorModel, tea.Cmd) {
	key := msg.String()

	// Fullscreen toggle (in-editor keybind).
	if key == "ctrl+f" {
		return e, func() tea.Msg { return editorToggleFullscreenMsg{} }
	}

	// Two-key pending: tt = toggle checkbox.
	if e.pendingT {
		e.pendingT = false
		if key == "t" {
			e.toggleCurrentTaskCheckbox()
		}
		return e, nil
	}

	if e.pendingOp != 0 {
		return e.updatePendingOperator(key)
	}
	if e.pendingPrefix != 0 {
		switch e.pendingPrefix {
		case 'g':
			e.pendingPrefix = 0
			switch key {
			case "g":
				e.moveTop()
				if e.renderMode {
					e.syncRenderViewTop()
				}
				return e, nil
			case "l":
				return e, func() tea.Msg { return editorLinkPickerMsg{} }
			case "u":
				return e, func() tea.Msg { return editorURLPromptMsg{} }
			}
		}
		e.pendingPrefix = 0
	}

	// Accumulate count digits (1-9 to start, 0-9 after first digit).
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		if key != "0" || e.pendingCount > 0 {
			e.pendingCount = e.pendingCount*10 + int(key[0]-'0')
			return e, nil
		}
	}

	switch key {
	case "h", "left":
		e.moveLeft()
	case "l", "right":
		e.moveRight()
	case "ctrl+left":
		e.moveToPoint(e.prevWordStartPoint())
	case "ctrl+right":
		e.moveToPoint(e.nextWordStartPoint())
	case "home":
		e.moveStartLine()
	case "end":
		e.moveEndLine()
	case "j", "down":
		n := max(1, e.pendingCount)
		for range n {
			e.moveDown()
		}
	case "k", "up":
		n := max(1, e.pendingCount)
		for range n {
			e.moveUp()
		}
	case "0":
		e.moveStartLine()
	case "^":
		e.moveFirstNonBlank()
	case "$":
		e.moveEndLine()
	case "w":
		e.moveToPoint(e.nextWordStartPoint())
	case "b":
		e.moveToPoint(e.prevWordStartPoint())
	case "e":
		e.moveToPoint(e.endOfWordPoint())
	case "G":
		e.moveBottom()
	case "g":
		e.pendingPrefix = 'g'
	case "t":
		e.pendingT = true
	case "i":
		e.mode = editorModeInsert
	case "a":
		if len(e.currentLine()) > 0 {
			e.col = min(e.col+1, len(e.currentLine()))
		}
		e.mode = editorModeInsert
	case "I":
		e.col = editorFirstNonBlank(e.currentLine())
		e.mode = editorModeInsert
	case "A":
		e.col = len(e.currentLine())
		e.mode = editorModeInsert
	case "o":
		e.openBelow()
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
	case "O":
		e.openAbove()
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
	case "v":
		e.renderMode = false
		e.mode = editorModeVisualChar
		e.visualAnchor = editorPoint{row: e.row, col: e.cursorDisplayCol()}
	case "V":
		e.renderMode = false
		e.mode = editorModeVisualLine
		e.visualAnchor = editorPoint{row: e.row, col: 0}
	case ":":
		e.mode = editorModeCmdline
		e.cmdBuf = nil
	case "/":
		e.mode = editorModeSearch
		e.searchBuf = nil
		e.lastSearchFwd = true
	case "?":
		e.mode = editorModeSearch
		e.searchBuf = nil
		e.lastSearchFwd = false
	case "d", "c", "y":
		e.pendingOp = rune(key[0])
	case "x":
		e.deleteChar()
	case "s":
		if len(e.currentLine()) > 0 {
			start := e.currentCharOffset()
			e.applyMotionRange(start, start+1, true, false)
		} else {
			e.mode = editorModeInsert
		}
	case "S":
		e.changeCurrentLine()
	case "p":
		e.pasteAfter()
	case "P":
		e.pasteBefore()
	case "u":
		e.undo()
	case "ctrl+r":
		e.redo()
	case "n":
		e.repeatSearch(false)
	case "N":
		e.repeatSearch(true)
	}
	e.pendingCount = 0
	e.ensureVisible()
	if e.renderMode {
		e.syncRenderViewTop()
	}
	return e, nil
}

func (e EditorModel) updateVisual(msg tea.KeyMsg) (EditorModel, tea.Cmd) {
	key := msg.String()
	if e.pendingPrefix == 'g' {
		e.pendingPrefix = 0
		switch key {
		case "l":
			return e, func() tea.Msg { return editorLinkPickerMsg{} }
		case "u":
			return e, func() tea.Msg { return editorURLPromptMsg{} }
		}
	}

	switch key {
	case "esc":
		e.mode = editorModeNormal
	case "h", "left":
		e.moveLeft()
	case "l", "right":
		e.moveRight()
	case "ctrl+left":
		e.moveToPoint(e.prevWordStartPoint())
	case "ctrl+right":
		e.moveToPoint(e.nextWordStartPoint())
	case "home":
		e.moveStartLine()
	case "end":
		e.moveEndLine()
	case "j", "down":
		e.moveDown()
	case "k", "up":
		e.moveUp()
	case "0":
		e.moveStartLine()
	case "^":
		e.moveFirstNonBlank()
	case "$":
		e.moveEndLine()
	case "w":
		e.moveToPoint(e.nextWordStartPoint())
	case "b":
		e.moveToPoint(e.prevWordStartPoint())
	case "e":
		e.moveToPoint(e.endOfWordPoint())
	case "G":
		e.moveBottom()
	case "g":
		e.pendingPrefix = 'g'
	case "v":
		if e.mode == editorModeVisualChar {
			e.mode = editorModeNormal
		} else {
			e.mode = editorModeVisualChar
			e.visualAnchor.col = e.cursorDisplayCol()
		}
	case "V":
		if e.mode == editorModeVisualLine {
			e.mode = editorModeNormal
		} else {
			e.mode = editorModeVisualLine
			e.visualAnchor.col = 0
		}
	case "d":
		e.deleteVisualSelection(false)
	case "c":
		e.deleteVisualSelection(true)
	case "y":
		e.yankVisualSelection()
	case "/":
		e.mode = editorModeSearch
		e.searchBuf = nil
		e.lastSearchFwd = true
	case "?":
		e.mode = editorModeSearch
		e.searchBuf = nil
		e.lastSearchFwd = false
	case ":":
		e.mode = editorModeCmdline
		e.cmdBuf = nil
	case "n":
		e.repeatSearch(false)
	case "N":
		e.repeatSearch(true)
	}
	e.ensureVisible()
	if e.mode == editorModeNormal {
		e.maybeRestoreRenderMode()
	}
	return e, nil
}

func (e EditorModel) updateInsert(msg tea.KeyMsg) (EditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		e.mode = editorModeNormal
		if len(e.currentLine()) > 0 && e.col > 0 {
			e.col--
		}
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
		e.ensureVisible()
		return e, nil
	case "enter":
		e.insertNewline()
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
		return e, nil
	case "backspace":
		e.backspace()
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
		return e, nil
	case "ctrl+w":
		e.deleteBackwardWord()
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
		return e, nil
	case "left":
		e.moveInsertLeft()
		return e, nil
	case "right":
		e.moveInsertRight()
		return e, nil
	case "ctrl+left":
		e.moveToPoint(e.prevWordStartPoint())
		return e, nil
	case "ctrl+right":
		e.moveToPoint(e.nextWordStartPoint())
		return e, nil
	case "home":
		e.col = 0
		return e, nil
	case "end":
		e.col = len(e.currentLine())
		return e, nil
	case "up":
		e.moveInsertUp()
		if e.renderMode {
			e.syncRenderViewTop()
		}
		return e, nil
	case "down":
		e.moveInsertDown()
		if e.renderMode {
			e.syncRenderViewTop()
		}
		return e, nil
	case "tab":
		e.insertText("\t")
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
		return e, nil
	case " ":
		e.insertText(" ")
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
		return e, nil
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		e.insertText(string(msg.Runes))
		if e.renderMode {
			e.recomputeRenderedDoc()
			e.syncRenderViewTop()
		}
	}
	return e, nil
}

func (e EditorModel) updateCmdline(msg tea.KeyMsg) (EditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		e.mode = editorModeNormal
		e.cmdBuf = nil
		return e, nil
	case "enter":
		return e, e.executeCmdline()
	case "backspace":
		if len(e.cmdBuf) > 0 {
			e.cmdBuf = e.cmdBuf[:len(e.cmdBuf)-1]
		}
		return e, nil
	case " ":
		e.cmdBuf = append(e.cmdBuf, ' ')
		return e, nil
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		e.cmdBuf = append(e.cmdBuf, msg.Runes...)
	}
	return e, nil
}

func (e EditorModel) updateSearch(msg tea.KeyMsg) (EditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		e.mode = editorModeNormal
		e.searchBuf = nil
		return e, nil
	case "enter":
		pattern := strings.TrimSpace(string(e.searchBuf))
		e.mode = editorModeNormal
		e.searchBuf = nil
		if pattern == "" {
			return e, nil
		}
		e.lastSearch = []rune(pattern)
		if !e.search(string(e.lastSearch), e.lastSearchFwd, true) {
			e.status = "pattern not found"
		}
		return e, nil
	case "backspace":
		if len(e.searchBuf) > 0 {
			e.searchBuf = e.searchBuf[:len(e.searchBuf)-1]
		}
		return e, nil
	case " ":
		e.searchBuf = append(e.searchBuf, ' ')
		return e, nil
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		e.searchBuf = append(e.searchBuf, msg.Runes...)
	}
	return e, nil
}

func (e *EditorModel) executeCmdline() tea.Cmd {
	cmd := strings.TrimSpace(string(e.cmdBuf))
	e.cmdBuf = nil
	e.mode = editorModeNormal

	switch cmd {
	case "q":
		if e.dirty {
			e.status = "no write since last change"
			return nil
		}
		return func() tea.Msg { return editorClosedMsg{} }
	case "q!":
		return func() tea.Msg { return editorClosedMsg{discarded: true} }
	case "w":
		return editorSaveCmd(e.rootDir, e.path, e.Content(), e.loadHash, e.passphrase, e.encrypted, false, false)
	case "w!":
		return editorSaveCmd(e.rootDir, e.path, e.Content(), e.loadHash, e.passphrase, e.encrypted, true, false)
	case "wq":
		return editorSaveCmd(e.rootDir, e.path, e.Content(), e.loadHash, e.passphrase, e.encrypted, false, true)
	case "e!":
		return editorReloadCmd(e.path, e.encrypted, e.passphrase)
	case "set relativenumber", "set rnu":
		e.relativeLineNumbers = true
		e.lineNumbersEnabled = true
		return func() tea.Msg { return editorRelativeNumbersMsg{enabled: true} }
	case "set norelativenumber", "set nornu":
		e.relativeLineNumbers = false
		return func() tea.Msg { return editorRelativeNumbersMsg{enabled: false} }
	default:
		// Numeric command: jump to line.
		n, err := strconv.Atoi(cmd)
		if err == nil && n >= 1 {
			e.row = clamp(n-1, 0, len(e.lines)-1)
			e.col = 0
			e.preferCol = 0
			e.ensureVisible()
			if e.renderMode {
				e.recomputeRenderedDoc()
				e.syncRenderViewTop()
			}
			return nil
		}
		e.status = "unknown command: :" + cmd
		return nil
	}
}

func (e *EditorModel) repeatSearch(reverse bool) {
	if len(e.lastSearch) == 0 {
		e.status = "no previous search"
		return
	}
	dir := e.lastSearchFwd
	if reverse {
		dir = !dir
	}
	if !e.search(string(e.lastSearch), dir, true) {
		e.status = "pattern not found"
	}
}

func (e *EditorModel) search(pattern string, forward, wrap bool) bool {
	needle := []rune(pattern)
	if len(needle) == 0 {
		return false
	}
	content := e.allRunes()
	if len(content) == 0 {
		return false
	}
	start := e.pointToOffset(editorPoint{row: e.row, col: e.cursorDisplayCol()})
	if forward {
		start++
	} else {
		start--
	}
	idx := editorFindRunes(content, needle, start, forward)
	if idx < 0 && wrap {
		if forward {
			idx = editorFindRunes(content, needle, 0, true)
		} else {
			idx = editorFindRunes(content, needle, len(content)-1, false)
		}
	}
	if idx < 0 {
		return false
	}
	e.setNormalCursorFromOffset(idx)
	e.lastSearchFwd = forward
	e.status = fmt.Sprintf("match: %s", pattern)
	return true
}

func (e *EditorModel) updatePendingOperator(key string) (EditorModel, tea.Cmd) {
	op := e.pendingOp
	e.pendingOp = 0
	e.pendingCount = 0
	switch op {
	case 'd':
		return e.applyOperator(key, false, false)
	case 'c':
		return e.applyOperator(key, true, false)
	case 'y':
		return e.applyOperator(key, false, true)
	default:
		return *e, nil
	}
}

func (e EditorModel) applyOperator(key string, change bool, yankOnly bool) (EditorModel, tea.Cmd) {
	switch key {
	case "d", "c", "y":
		switch key {
		case "d":
			e.deleteCurrentLine()
		case "c":
			e.changeCurrentLine()
		case "y":
			e.yankCurrentLine()
		}
		return e, nil
	case "w":
		e.applyMotionRange(e.currentCharOffset(), e.pointToOffset(e.nextWordStartPoint()), change, yankOnly)
	case "b":
		e.applyMotionRange(e.pointToOffset(e.prevWordStartPoint()), e.pointToOffset(editorPoint{row: e.row, col: e.cursorDisplayCol()})+1, change, yankOnly)
	case "e":
		end := e.pointToOffset(e.pointAfterChar(e.endOfWordPoint()))
		e.applyMotionRange(e.currentCharOffset(), end, change, yankOnly)
	case "0":
		e.applyMotionRange(e.pointToOffset(editorPoint{row: e.row, col: 0}), e.currentCharOffset()+1, change, yankOnly)
	case "$":
		e.applyMotionRange(e.currentCharOffset(), e.lineEndOffset(e.row), change, yankOnly)
	}
	return e, nil
}

func (e *EditorModel) applyMotionRange(start, end int, change bool, yankOnly bool) {
	if end < start {
		start, end = end, start
	}
	if start == end {
		e.status = "nothing to change"
		return
	}
	text := e.substringByOffsets(start, end)
	e.registers = editorRegister{text: text}
	if yankOnly {
		e.status = "yanked"
		return
	}
	e.replaceOffsets(start, end, "", start, change)
	if change {
		e.status = "changed"
	} else {
		e.status = "deleted"
	}
}

func (e *EditorModel) moveToPoint(p editorPoint) {
	e.row = clamp(p.row, 0, len(e.lines)-1)
	maxCol := e.normalMaxCol(e.row)
	e.col = clamp(p.col, 0, maxCol)
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveLeft() {
	if len(e.currentLine()) == 0 {
		e.col = 0
		return
	}
	e.col = max(0, e.col-1)
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveRight() {
	e.col = min(e.normalMaxCol(e.row), e.col+1)
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveUp() {
	if e.row > 0 {
		e.row--
		e.col = clamp(e.preferCol, 0, e.normalMaxCol(e.row))
	}
	e.ensureVisible()
}

func (e *EditorModel) moveDown() {
	if e.row < len(e.lines)-1 {
		e.row++
		e.col = clamp(e.preferCol, 0, e.normalMaxCol(e.row))
	}
	e.ensureVisible()
}

func (e *EditorModel) moveStartLine() {
	e.col = 0
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveFirstNonBlank() {
	e.col = editorFirstNonBlank(e.currentLine())
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveEndLine() {
	e.col = e.normalMaxCol(e.row)
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveTop() {
	e.row = 0
	e.col = clamp(e.col, 0, e.normalMaxCol(0))
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveBottom() {
	e.row = len(e.lines) - 1
	e.col = clamp(e.col, 0, e.normalMaxCol(e.row))
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveInsertLeft() {
	e.col = max(0, e.col-1)
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveInsertRight() {
	e.col = min(len(e.currentLine()), e.col+1)
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) moveInsertUp() {
	if e.row > 0 {
		e.row--
		e.col = clamp(e.preferCol, 0, len(e.currentLine()))
	}
	e.ensureVisible()
}

func (e *EditorModel) moveInsertDown() {
	if e.row < len(e.lines)-1 {
		e.row++
		e.col = clamp(e.preferCol, 0, len(e.currentLine()))
	}
	e.ensureVisible()
}

func (e EditorModel) nextWordStartPoint() editorPoint {
	return e.offsetToNormalPoint(editorNextWordStart(e.allRunes(), e.currentCharOffset()))
}

func (e EditorModel) prevWordStartPoint() editorPoint {
	return e.offsetToNormalPoint(editorPrevWordStart(e.allRunes(), e.currentCharOffset()))
}

func (e EditorModel) endOfWordPoint() editorPoint {
	return e.offsetToNormalPoint(editorEndOfWord(e.allRunes(), e.currentCharOffset()))
}

func (e *EditorModel) insertText(text string) {
	start := e.insertOffset()
	e.replaceOffsets(start, start, text, start+len([]rune(text)), true)
}

func (e *EditorModel) insertNewline() {
	start := e.insertOffset()
	e.replaceOffsets(start, start, "\n\n", start+2, true)
}

func (e *EditorModel) backspace() {
	start := e.insertOffset()
	if start == 0 {
		return
	}
	e.replaceOffsets(start-1, start, "", start-1, true)
}

func (e *EditorModel) deleteBackwardWord() {
	end := e.insertOffset()
	if end == 0 {
		return
	}
	start := editorPrevWordStart(e.allRunes(), end)
	e.replaceOffsets(start, end, "", start, true)
}

func (e *EditorModel) deleteChar() {
	if len(e.currentLine()) == 0 {
		return
	}
	start := e.currentCharOffset()
	e.applyMotionRange(start, start+1, false, false)
}

func (e *EditorModel) openBelow() {
	e.pushUndo()
	insertAt := e.row + 1
	blank := [][]rune{{}, {}}
	e.lines = append(e.lines[:insertAt], append(blank, e.lines[insertAt:]...)...)
	e.row = insertAt + 1
	e.col = 0
	e.preferCol = 0
	e.mode = editorModeInsert
	e.dirty = true
	e.ensureVisible()
}

func (e *EditorModel) openAbove() {
	e.pushUndo()
	insertAt := e.row
	blank := [][]rune{{}, {}}
	e.lines = append(e.lines[:insertAt], append(blank, e.lines[insertAt:]...)...)
	e.row = insertAt
	e.col = 0
	e.preferCol = 0
	e.mode = editorModeInsert
	e.dirty = true
	e.ensureVisible()
}

func (e *EditorModel) yankCurrentLine() {
	start, end := e.selectedLineOffsets(e.row, e.row)
	e.registers = editorRegister{text: e.substringByOffsets(start, end), linewise: true}
	e.status = "yanked line"
}

func (e *EditorModel) deleteCurrentLine() {
	start, end := e.selectedLineOffsets(e.row, e.row)
	e.registers = editorRegister{text: e.substringByOffsets(start, end), linewise: true}
	e.replaceLinewise(start, end, false)
	e.status = "deleted line"
}

func (e *EditorModel) changeCurrentLine() {
	start, end := e.selectedLineOffsets(e.row, e.row)
	e.registers = editorRegister{text: e.substringByOffsets(start, end), linewise: true}
	e.replaceLinewise(start, end, true)
	e.status = "changed line"
}

func (e *EditorModel) yankVisualSelection() {
	var text string
	if e.mode == editorModeVisualLine {
		startRow, endRow := e.selectedLineBounds()
		start, end := e.selectedLineOffsets(startRow, endRow)
		text = e.substringByOffsets(start, end)
		e.registers = editorRegister{text: text, linewise: true}
	} else {
		start, end, ok := e.visualSelectionOffsets()
		if !ok {
			return
		}
		text = e.substringByOffsets(start, end)
		e.registers = editorRegister{text: text}
	}
	_ = writeClipboard(text)
	e.mode = editorModeNormal
	e.status = "yanked"
}

func (e *EditorModel) deleteVisualSelection(change bool) {
	if e.mode == editorModeVisualLine {
		startRow, endRow := e.selectedLineBounds()
		start, end := e.selectedLineOffsets(startRow, endRow)
		e.registers = editorRegister{text: e.substringByOffsets(start, end), linewise: true}
		e.replaceLinewise(start, end, change)
	} else {
		start, end, ok := e.visualSelectionOffsets()
		if !ok {
			return
		}
		e.registers = editorRegister{text: e.substringByOffsets(start, end)}
		e.replaceOffsets(start, end, "", start, change)
	}
	if change {
		e.status = "changed"
	} else {
		e.status = "deleted"
	}
}

func (e *EditorModel) pasteAfter() {
	e.paste(true)
}

func (e *EditorModel) pasteBefore() {
	e.paste(false)
}

func (e *EditorModel) paste(after bool) {
	if e.registers.text == "" {
		return
	}
	if e.registers.linewise {
		e.pushUndo()
		lines := editorSplitRunes([]rune(e.registers.text))
		insertAt := e.row
		if after {
			insertAt++
		}
		e.lines = append(e.lines[:insertAt], append(lines, e.lines[insertAt:]...)...)
		e.row = insertAt
		e.col = 0
		e.preferCol = 0
		e.dirty = true
		e.mode = editorModeNormal
		e.ensureVisible()
		return
	}
	offset := e.currentCharOffset()
	if after && len(e.currentLine()) > 0 {
		offset++
	}
	e.replaceOffsets(offset, offset, e.registers.text, offset+len([]rune(e.registers.text))-1, false)
}

func (e *EditorModel) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	e.redoStack = append(e.redoStack, e.snapshot())
	s := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.restoreSnapshot(s)
	e.status = "undo"
	if e.renderMode {
		e.recomputeRenderedDoc()
		e.syncRenderViewTop()
	}
}

func (e *EditorModel) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	e.undoStack = append(e.undoStack, e.snapshot())
	s := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.restoreSnapshot(s)
	e.status = "redo"
	if e.renderMode {
		e.recomputeRenderedDoc()
		e.syncRenderViewTop()
	}
}

func (e *EditorModel) pushUndo() {
	e.undoStack = append(e.undoStack, e.snapshot())
	if len(e.undoStack) > editorUndoLimit {
		e.undoStack = e.undoStack[len(e.undoStack)-editorUndoLimit:]
	}
	e.redoStack = nil
}

func (e EditorModel) snapshot() editorUndoSnapshot {
	return editorUndoSnapshot{
		lines:     editorCloneLines(e.lines),
		row:       e.row,
		col:       e.col,
		preferCol: e.preferCol,
		viewTop:   e.viewTop,
		viewLeft:  e.viewLeft,
	}
}

func (e *EditorModel) restoreSnapshot(s editorUndoSnapshot) {
	e.lines = editorCloneLines(s.lines)
	e.row = s.row
	e.col = s.col
	e.preferCol = s.preferCol
	e.viewTop = s.viewTop
	e.viewLeft = s.viewLeft
	e.mode = editorModeNormal
	e.pendingPrefix = 0
	e.pendingOp = 0
	e.pendingCount = 0
	e.dirty = editorHashContent(e.Content()) != e.loadHash
	e.ensureVisible()
}

func (e *EditorModel) replaceOffsets(start, end int, replacement string, cursorOff int, enterInsert bool) {
	start = clamp(start, 0, len(e.allRunes()))
	end = clamp(end, start, len(e.allRunes()))
	e.pushUndo()
	all := e.allRunes()
	out := append([]rune{}, all[:start]...)
	out = append(out, []rune(replacement)...)
	out = append(out, all[end:]...)
	e.setFromRunes(out)
	if enterInsert {
		e.mode = editorModeInsert
		e.setInsertCursorFromOffset(cursorOff)
	} else {
		e.mode = editorModeNormal
		e.setNormalCursorFromOffset(max(0, cursorOff-1))
	}
	e.pendingPrefix = 0
	e.pendingOp = 0
	e.pendingCount = 0
	e.dirty = true
	e.ensureVisible()
	if e.renderMode {
		e.recomputeRenderedDoc()
		e.syncRenderViewTop()
	}
}

func (e *EditorModel) replaceLinewise(start, end int, enterInsert bool) {
	start = clamp(start, 0, len(e.allRunes()))
	end = clamp(end, start, len(e.allRunes()))
	replacement := ""
	cursor := start
	if enterInsert {
		replacement = "\n"
	}
	e.pushUndo()
	all := e.allRunes()
	out := append([]rune{}, all[:start]...)
	out = append(out, []rune(replacement)...)
	out = append(out, all[end:]...)
	e.setFromRunes(out)
	if enterInsert {
		e.mode = editorModeInsert
		e.setInsertCursorFromOffset(cursor)
	} else {
		e.mode = editorModeNormal
		e.setNormalCursorFromOffset(cursor)
	}
	e.pendingPrefix = 0
	e.pendingOp = 0
	e.pendingCount = 0
	e.dirty = true
	e.ensureVisible()
	if e.renderMode {
		e.recomputeRenderedDoc()
		e.syncRenderViewTop()
	}
}

func (e EditorModel) currentLine() []rune {
	if e.row < 0 || e.row >= len(e.lines) {
		return nil
	}
	return e.lines[e.row]
}

func (e EditorModel) normalMaxCol(row int) int {
	if row < 0 || row >= len(e.lines) {
		return 0
	}
	if len(e.lines[row]) == 0 {
		return 0
	}
	return len(e.lines[row]) - 1
}

func (e *EditorModel) ensureVisible() {
	if e.row < e.viewTop {
		e.viewTop = e.row
	}
	if e.row >= e.viewTop+e.contentHeight {
		e.viewTop = e.row - e.contentHeight + 1
	}
	if e.viewTop < 0 {
		e.viewTop = 0
	}
	cursorCol := e.cursorDisplayCol()
	if cursorCol < e.viewLeft {
		e.viewLeft = cursorCol
	}
	if cursorCol >= e.viewLeft+e.width {
		e.viewLeft = cursorCol - e.width + 1
	}
	if e.viewLeft < 0 {
		e.viewLeft = 0
	}
}

func (e EditorModel) currentCharOffset() int {
	return e.pointToOffset(editorPoint{row: e.row, col: e.cursorDisplayCol()})
}

func (e EditorModel) insertOffset() int {
	if e.mode == editorModeInsert {
		return e.pointToOffset(editorPoint{row: e.row, col: clamp(e.col, 0, len(e.currentLine()))})
	}
	return e.currentCharOffset()
}

func (e EditorModel) pointToOffset(p editorPoint) int {
	if len(e.lines) == 0 {
		return 0
	}
	p.row = clamp(p.row, 0, len(e.lines)-1)
	offset := 0
	for i := 0; i < p.row; i++ {
		offset += len(e.lines[i]) + 1
	}
	col := clamp(p.col, 0, len(e.lines[p.row]))
	offset += col
	return offset
}

func (e EditorModel) offsetToInsertPoint(offset int) editorPoint {
	offset = clamp(offset, 0, len(e.allRunes()))
	running := 0
	for row, line := range e.lines {
		lineLen := len(line)
		if offset <= running+lineLen {
			return editorPoint{row: row, col: offset - running}
		}
		running += lineLen + 1
	}
	last := len(e.lines) - 1
	return editorPoint{row: last, col: len(e.lines[last])}
}

func (e EditorModel) offsetToNormalPoint(offset int) editorPoint {
	p := e.offsetToInsertPoint(offset)
	if len(e.lines[p.row]) == 0 {
		p.col = 0
		return p
	}
	if p.col >= len(e.lines[p.row]) {
		p.col = len(e.lines[p.row]) - 1
	}
	return p
}

func (e *EditorModel) setInsertCursorFromOffset(offset int) {
	p := e.offsetToInsertPoint(offset)
	e.row, e.col = p.row, p.col
	e.preferCol = e.col
	e.ensureVisible()
}

func (e *EditorModel) setNormalCursorFromOffset(offset int) {
	p := e.offsetToNormalPoint(offset)
	e.row, e.col = p.row, p.col
	e.preferCol = e.col
	e.ensureVisible()
}

func (e EditorModel) pointAfterChar(p editorPoint) editorPoint {
	if p.row < 0 || p.row >= len(e.lines) {
		return editorPoint{}
	}
	line := e.lines[p.row]
	if len(line) == 0 {
		// Empty line: advance past the implicit newline to include it in selections.
		if p.row+1 < len(e.lines) {
			return editorPoint{row: p.row + 1, col: 0}
		}
		return editorPoint{row: p.row, col: 0}
	}
	if p.col+1 <= len(line) {
		return editorPoint{row: p.row, col: p.col + 1}
	}
	return editorPoint{row: p.row, col: len(line)}
}

func (e EditorModel) visualSelectionOffsets() (int, int, bool) {
	if e.mode != editorModeVisualChar {
		return 0, 0, false
	}
	start := e.visualAnchor
	end := editorPoint{row: e.row, col: e.cursorDisplayCol()}
	if editorComparePoint(start, end) > 0 {
		start, end = end, start
	}
	return e.pointToOffset(start), e.pointToOffset(e.pointAfterChar(end)), true
}

func (e EditorModel) selectedLineBounds() (int, int) {
	start, end := e.visualAnchor.row, e.row
	if start > end {
		start, end = end, start
	}
	return start, end
}

func (e EditorModel) selectedLineOffsets(startRow, endRow int) (int, int) {
	start := e.pointToOffset(editorPoint{row: startRow, col: 0})
	if endRow+1 < len(e.lines) {
		return start, e.pointToOffset(editorPoint{row: endRow + 1, col: 0})
	}
	return start, len(e.allRunes())
}

func (e EditorModel) substringByOffsets(start, end int) string {
	all := e.allRunes()
	start = clamp(start, 0, len(all))
	end = clamp(end, start, len(all))
	return string(all[start:end])
}

func (e *EditorModel) setFromRunes(content []rune) {
	e.lines = editorSplitRunes(content)
	if len(e.lines) == 0 {
		e.lines = [][]rune{{}}
	}
}

func (e EditorModel) allRunes() []rune {
	return []rune(editorJoinLines(e.lines))
}

func (e EditorModel) lineEndOffset(row int) int {
	if row < 0 || row >= len(e.lines) {
		return len(e.allRunes())
	}
	return e.pointToOffset(editorPoint{row: row, col: len(e.lines[row])})
}

func (e *EditorModel) InsertWikilink(target string) {
	target = strings.TrimSpace(target)
	if target == "" {
		e.status = "no target selected"
		return
	}
	label := ""
	if e.mode == editorModeVisualChar || e.mode == editorModeVisualLine {
		label = strings.Join(strings.Fields(e.visualSelectionText()), " ")
	}
	replacement := "[[" + target + "]]"
	if label != "" && label != target {
		replacement = "[[" + target + "|" + label + "]]"
	}
	e.insertLinkReplacement(replacement)
	e.status = "inserted note link"
}

func (e *EditorModel) InsertURLLink(url string) {
	url = strings.TrimSpace(url)
	if url == "" {
		e.status = "URL cannot be empty"
		return
	}
	if e.mode == editorModeVisualChar || e.mode == editorModeVisualLine {
		label := strings.Join(strings.Fields(e.visualSelectionText()), " ")
		if label == "" {
			label = "link"
		}
		e.insertLinkReplacement("[" + label + "](" + url + ")")
		e.status = "inserted URL link"
		return
	}
	replacement := "[label](" + url + ")"
	start := e.insertOffset()
	e.replaceOffsets(start, start, replacement, start+1, true)
	e.status = "inserted URL link"
}

func (e *EditorModel) insertLinkReplacement(replacement string) {
	if e.mode == editorModeVisualLine {
		startRow, endRow := e.selectedLineBounds()
		start, end := e.selectedLineOffsets(startRow, endRow)
		e.replaceOffsets(start, end, replacement, start+len([]rune(replacement)), false)
		return
	}
	if start, end, ok := e.visualSelectionOffsets(); ok {
		e.replaceOffsets(start, end, replacement, start+len([]rune(replacement)), false)
		return
	}
	start := e.insertOffset()
	e.replaceOffsets(start, start, replacement, start+len([]rune(replacement)), false)
}

func (e EditorModel) visualSelectionText() string {
	if e.mode == editorModeVisualLine {
		startRow, endRow := e.selectedLineBounds()
		start, end := e.selectedLineOffsets(startRow, endRow)
		return e.substringByOffsets(start, end)
	}
	start, end, ok := e.visualSelectionOffsets()
	if !ok {
		return ""
	}
	return e.substringByOffsets(start, end)
}

func editorLoadCmd(rootDir, path, relPath string, encrypted bool, passphrase string, isTemp bool) tea.Cmd {
	return func() tea.Msg {
		raw, err := notes.ReadAll(path)
		if err != nil {
			return editorLoadedMsg{err: err}
		}
		content := raw
		if encrypted {
			content, err = notes.PrepareForEdit(raw, passphrase)
			if err != nil {
				return editorLoadedMsg{err: err}
			}
		}
		info, err := os.Stat(path)
		if err != nil {
			return editorLoadedMsg{err: err}
		}
		return editorLoadedMsg{
			path:      path,
			relPath:   relPath,
			content:   content,
			hash:      editorHashContent(raw),
			modTime:   info.ModTime(),
			encrypted: encrypted,
			isTemp:    isTemp,
		}
	}
}

func saveNoteVersionAndEditorLoadCmd(rootDir, path, relPath, passphrase string, isTemp bool) tea.Cmd {
	return func() tea.Msg {
		raw, err := notes.ReadAll(path)
		if err != nil {
			return editorLoadedMsg{err: err}
		}
		if rel, relErr := filepath.Rel(rootDir, path); relErr == nil {
			_ = notes.SaveVersion(rootDir, filepath.ToSlash(rel), raw)
		}
		content, err := notes.PrepareForEdit(raw, passphrase)
		if err != nil {
			return editorLoadedMsg{err: err}
		}
		info, err := os.Stat(path)
		if err != nil {
			return editorLoadedMsg{err: err}
		}
		return editorLoadedMsg{
			path:      path,
			relPath:   relPath,
			content:   content,
			hash:      editorHashContent(raw),
			modTime:   info.ModTime(),
			encrypted: true,
			isTemp:    isTemp,
		}
	}
}

func editorSaveCmd(rootDir, path, content, expectedHash, passphrase string, encrypted, force, closeAfter bool) tea.Cmd {
	return func() tea.Msg {
		raw, err := notes.ReadAll(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return editorSaveErrMsg{err: err}
		}
		if !force {
			if err != nil || editorHashContent(raw) != expectedHash {
				return editorConflictMsg{}
			}
		}

		if encrypted && err == nil {
			if relPath, relErr := filepath.Rel(rootDir, path); relErr == nil {
				_ = notes.SaveVersion(rootDir, filepath.ToSlash(relPath), raw)
			}
		}

		newPath := path
		if encrypted {
			newPath, err = notes.ReencryptBody(path, content, passphrase)
		} else {
			if err = notes.WriteAll(path, content); err == nil {
				var renamed bool
				newPath, renamed, err = notes.RenameFromTitle(path)
				_ = renamed
			}
		}
		if err != nil {
			return editorSaveErrMsg{err: err}
		}
		if strings.TrimSpace(newPath) == "" {
			return editorSavedMsg{oldPath: path, discarded: true, closeAfter: true}
		}
		finalRaw, err := notes.ReadAll(newPath)
		if err != nil {
			return editorSaveErrMsg{err: err}
		}
		info, err := os.Stat(newPath)
		if err != nil {
			return editorSaveErrMsg{err: err}
		}
		return editorSavedMsg{
			oldPath:    path,
			newPath:    newPath,
			hash:       editorHashContent(finalRaw),
			modTime:    info.ModTime(),
			closeAfter: closeAfter,
		}
	}
}

func editorReloadCmd(path string, encrypted bool, passphrase string) tea.Cmd {
	return func() tea.Msg {
		raw, err := notes.ReadAll(path)
		if err != nil {
			return editorReloadErrMsg{err: err}
		}
		content := raw
		if encrypted {
			content, err = notes.PrepareForEdit(raw, passphrase)
			if err != nil {
				return editorReloadErrMsg{err: err}
			}
		}
		info, err := os.Stat(path)
		if err != nil {
			return editorReloadErrMsg{err: err}
		}
		return editorReloadedMsg{
			content: content,
			hash:    editorHashContent(raw),
			modTime: info.ModTime(),
		}
	}
}

func editorNormalizeContent(content string) []rune {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if content == "" {
		return nil
	}
	return []rune(content)
}

func editorSplitRunes(content []rune) [][]rune {
	if len(content) == 0 {
		return [][]rune{{}}
	}
	lines := make([][]rune, 0, strings.Count(string(content), "\n")+1)
	start := 0
	for i, r := range content {
		if r == '\n' {
			lines = append(lines, append([]rune{}, content[start:i]...))
			start = i + 1
		}
	}
	lines = append(lines, append([]rune{}, content[start:]...))
	return lines
}

func editorJoinLines(lines [][]rune) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(string(line))
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func editorCloneLines(lines [][]rune) [][]rune {
	out := make([][]rune, len(lines))
	for i, line := range lines {
		out[i] = append([]rune{}, line...)
	}
	return out
}

func editorHashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func editorRelativePath(rootDir, path string, isTemp bool) string {
	var base string
	if isTemp {
		base = notes.TempRoot(rootDir)
	} else {
		base = rootDir
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}

func editorComparePoint(a, b editorPoint) int {
	if a.row != b.row {
		if a.row < b.row {
			return -1
		}
		return 1
	}
	if a.col < b.col {
		return -1
	}
	if a.col > b.col {
		return 1
	}
	return 0
}

func editorFirstNonBlank(line []rune) int {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return 0
}

func editorFillLine(text string, width int, bg, fg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Width(width).
		Foreground(fg).
		Background(bg).
		Render(trimOrPad(text, width))
}

func editorFindRunes(haystack, needle []rune, start int, forward bool) int {
	if len(needle) == 0 || len(haystack) == 0 {
		return -1
	}
	if forward {
		start = clamp(start, 0, len(haystack)-1)
		for i := start; i <= len(haystack)-len(needle); i++ {
			if editorHasPrefix(haystack[i:], needle) {
				return i
			}
		}
		return -1
	}
	start = clamp(start, 0, len(haystack)-1)
	for i := start; i >= 0; i-- {
		if i+len(needle) > len(haystack) {
			continue
		}
		if editorHasPrefix(haystack[i:], needle) {
			return i
		}
	}
	return -1
}

func editorHasPrefix(haystack, needle []rune) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := range needle {
		if haystack[i] != needle[i] {
			return false
		}
	}
	return true
}

func editorIsWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func editorNextWordStart(content []rune, offset int) int {
	if len(content) == 0 {
		return 0
	}
	offset = clamp(offset, 0, len(content)-1)
	i := offset
	if editorIsWordRune(content[i]) {
		for i < len(content) && editorIsWordRune(content[i]) {
			i++
		}
	}
	for i < len(content) && !editorIsWordRune(content[i]) {
		i++
	}
	if i >= len(content) {
		return len(content) - 1
	}
	return i
}

func editorPrevWordStart(content []rune, offset int) int {
	if len(content) == 0 {
		return 0
	}
	offset = clamp(offset, 0, len(content)-1)
	i := offset - 1
	if i < 0 {
		return 0
	}
	for i >= 0 && !editorIsWordRune(content[i]) {
		i--
	}
	for i > 0 && editorIsWordRune(content[i-1]) {
		i--
	}
	if i < 0 {
		return 0
	}
	return i
}

func editorEndOfWord(content []rune, offset int) int {
	if len(content) == 0 {
		return 0
	}
	offset = clamp(offset, 0, len(content)-1)
	i := offset
	if !editorIsWordRune(content[i]) {
		i = editorNextWordStart(content, offset)
	}
	for i+1 < len(content) && editorIsWordRune(content[i+1]) {
		i++
	}
	return i
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
