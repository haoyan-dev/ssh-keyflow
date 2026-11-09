package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func assertPanesFit(t *testing.T, m model) {
	t.Helper()
	assertFits(t, m)
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if len(lines) != m.height || !strings.HasPrefix(lines[0], "╭") || !strings.HasPrefix(lines[len(lines)-1], "╰") {
		t.Fatalf("panes are cut off at %dx%d:\n%s", m.width, m.height, strings.Join(lines, "\n"))
	}
	for _, line := range lines {
		if ansi.StringWidth(line) != m.width {
			t.Fatalf("pane edge misaligned: %q", line)
		}
	}
}

func TestPanesFitAcrossSizesAndInteractionStates(t *testing.T) {
	for _, size := range [][2]int{{32, 12}, {40, 16}, {60, 20}, {80, 24}, {120, 40}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m := fixture(t)
			m = update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			assertPanesFit(t, m)
			m.status = strings.Repeat("Long 日本語 message ", 40)
			for _, key := range []rune{'/', tea.KeyEnter, 'x', tea.KeyEnter, 'v', '?', tea.KeyEscape, tea.KeyEscape, tea.KeyEscape, 'i'} {
				m = update(m, press(key))
				assertPanesFit(t, m)
			}
			for i := 0; i < len(m.fields)+3; i++ {
				assertPanesFit(t, m)
				m = update(m, press(tea.KeyTab))
			}
			m.formError = strings.Repeat("Invalid path 日本語 ", 40)
			m.renderForm()
			assertPanesFit(t, m)
			m = update(m, press(tea.KeyEscape), press(tea.KeyTab), press('g'))
			assertPanesFit(t, m)
			m = update(m, press(tea.KeyEnter))
			assertPanesFit(t, m)
			m = update(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
			assertPanesFit(t, m)
			m = update(m, press('?'))
			assertPanesFit(t, m)
			m.page = "result"
			m.setDocument(strings.Repeat("Result output 日本語\n", 40))
			assertPanesFit(t, m)
		})
	}
}

func TestSearchAndMessagesHaveSeparatePanesAtMinimumSize(t *testing.T) {
	m := fixture(t)
	m = update(m, tea.WindowSizeMsg{Width: 32, Height: 12}, press('/'), text("日本語"))
	m.stale = true
	assertPanesFit(t, m)
	screen := ansi.Strip(m.View().Content)
	for _, want := range []string{" Search ", " Keys ", " Messages", " Menu ", "F5 refresh"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing pane or recovery %q:\n%s", want, screen)
		}
	}
	cursor := m.View().Cursor
	if cursor == nil || cursor.Y != 3 || cursor.X < 2 || cursor.X >= m.width-2 {
		t.Fatalf("search cursor is outside its pane: %+v", cursor)
	}
	m = update(m, press(tea.KeyEnter))
	assertPanesFit(t, m)
	if m.View().Cursor != nil {
		t.Fatal("applied search retained an editing cursor")
	}
}

func TestLongInputsKeepCaretInsidePaneWithoutFalseTruncation(t *testing.T) {
	m := fixture(t)
	m = update(m, tea.WindowSizeMsg{Width: 32, Height: 12}, press('/'), text("short"))
	if strings.Contains(strings.Split(ansi.Strip(m.View().Content), "\n")[3], "…") {
		t.Fatal("short search falsely indicates truncation")
	}
	for _, value := range []string{strings.Repeat("a", 120), strings.Repeat("日本語", 40)} {
		m = update(m, text(value))
		assertPanesFit(t, m)
		cursor := m.View().Cursor
		if cursor == nil || cursor.X >= m.width-2 {
			t.Fatalf("long search caret reached pane border: %+v", cursor)
		}
	}
	m = update(m, press(tea.KeyEscape), press('x'), press('i'), text(strings.Repeat("/long/path", 40)))
	assertPanesFit(t, m)
	cursor := m.View().Cursor
	if cursor == nil || cursor.X >= m.width-2 {
		t.Fatalf("long form caret reached pane border: %+v", cursor)
	}
}

func TestEntrySpacingAndPagingKeepSelectedRecordVisible(t *testing.T) {
	m := fixture(t)
	key := m.view.State.Keys[0]
	for i := 2; i <= 30; i++ {
		copy := key
		copy.ID, copy.Name = fmt.Sprintf("key-%02d", i), fmt.Sprintf("Entry %02d", i)
		m.view.State.Keys = append(m.view.State.Keys, copy)
	}
	screen := strings.Split(ansi.Strip(m.listText(m.entries(), 0)), "\n")
	if len(screen) < 4 || screen[2] != "" || !strings.HasPrefix(screen[3], "  Entry 02") {
		t.Fatalf("entries have no blank separator:\n%s", strings.Join(screen, "\n"))
	}
	for _, size := range [][2]int{{32, 12}, {80, 24}} {
		m = update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, key := range []rune{tea.KeyHome, tea.KeyPgDown, tea.KeyPgDown, tea.KeyEnd, tea.KeyPgUp, tea.KeyUp} {
			m = update(m, press(key))
			assertPanesFit(t, m)
			target := m.visibleTarget()
			rendered := ansi.Strip(m.View().Content)
			if !strings.Contains(rendered, "> "+target.title) {
				t.Fatalf("selection disappeared after paging: %s\n%s", target.title, rendered)
			}
		}
	}
}

func TestFormCaretTracksInputPaneAndButtonsStayInMenu(t *testing.T) {
	m := fixture(t)
	m.tab = 1
	m.beginForm("add")
	for _, size := range [][2]int{{80, 24}, {32, 12}, {60, 20}} {
		m = update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for i := 0; i < len(m.fields)+3; i++ {
			m = update(m, press(tea.KeyTab))
			assertPanesFit(t, m)
			screen := ansi.Strip(m.View().Content)
			_, menu, _ := strings.Cut(screen, " Menu ")
			if !strings.Contains(strings.ReplaceAll(menu, ">[", "["), "[Continue] [Cancel] [Help]") {
				t.Fatalf("form buttons scrolled out of menu:\n%s", screen)
			}
			cursor := m.View().Cursor
			if m.focus >= len(m.fields) {
				if cursor != nil {
					t.Fatal("menu button has a text caret")
				}
				continue
			}
			if cursor == nil || cursor.Y < m.bodyTop() || cursor.Y >= m.bodyTop()+m.content.Height() || cursor.X < 2 || cursor.X >= m.width-2 {
				t.Fatalf("focused input caret is outside pane: focus=%d cursor=%+v\n%s", m.focus, cursor, screen)
			}
			line := strings.Split(screen, "\n")[cursor.Y]
			if m.fields[m.focus].input.Value() == "" && strings.TrimSpace(line[3:len(line)-3]) != "" {
				t.Fatalf("caret is on a label instead of the empty input: %q", line)
			}
		}
	}
}

func TestColorHasPlainTextCuesAndHonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	m := fixture(t)
	colored := m.View().Content
	if !strings.Contains(colored, "\x1b[") {
		t.Fatal("interface has no color or emphasis")
	}
	plain := ansi.Strip(colored)
	for _, cue := range []string{"[Keys]", "> 日本語 key", "Messages", "Menu"} {
		if !strings.Contains(plain, cue) {
			t.Fatalf("state relies only on color: missing %q", cue)
		}
	}
	t.Setenv("NO_COLOR", "1")
	if got := m.View().Content; got != plain {
		t.Fatalf("NO_COLOR changed layout or left escapes:\n%q", got)
	}
	m = update(m, press('i'))
	if strings.Contains(m.View().Content, "\x1b") {
		t.Fatal("NO_COLOR left colored form input")
	}
}

func TestPresentationAllowsOnlyTrustedSGR(t *testing.T) {
	m := fixture(t)
	payload := "safe\x1b[2J\x1b]52;c;c2VjcmV0\a\x7f\n\tanother line"
	m.view.State.Keys[0].Name = payload
	m.status = payload
	m = update(m, press('/'), text("safe"), press(tea.KeyEnter))
	// Strip only SGR generated by the UI, not arbitrary terminal controls.
	sgr := regexp.MustCompile(`\x1b\[[0-9;:]*m`)
	check := func() {
		t.Helper()
		rendered := sgr.ReplaceAllString(m.View().Content, "")
		if strings.ContainsAny(rendered, "\x1b\x7f\a") {
			t.Fatalf("untrusted terminal controls reached a pane: %q", rendered)
		}
	}
	check()
	assertPanesFit(t, m)
	m = update(m, press(tea.KeyEnter))
	check()
	m = update(m, press(tea.KeyEscape), press('i'), text(payload))
	check()
}
