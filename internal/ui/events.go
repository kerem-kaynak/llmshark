package ui

import (
	"context"
	"errors"
	"fmt"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kerem-kaynak/llmshark/internal/markdown"
	"github.com/kerem-kaynak/llmshark/internal/postgres"
	"github.com/kerem-kaynak/llmshark/internal/storage"
)

type errMsg struct {
	error
}

type credsMsg struct {
	creds *storage.Credentials
}

type noCredsMsg struct{}

type schemasMsg struct {
	schemas []postgres.Schema
}

type cursorPosition struct {
	itemType string
	schema   int
	table    int
	column   int
}

func (m *model) deselectAll() {
	for si := range m.schemas {
		schema := &m.schemas[si]
		schema.Selected = false
		for ti := range schema.Tables {
			table := &schema.Tables[ti]
			table.Selected = false
			for ci := range table.Columns {
				table.Columns[ci].Selected = false
			}
		}
	}
}

func (m model) updateCredentials(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear error on any key press
		m.err = nil

		switch msg.String() {
		case "enter":
			// Clear existing data before attempting new connection
			m.schemas = nil
			m.client = nil

			creds := &storage.Credentials{
				Host:     m.inputs[0].Value(),
				Port:     m.inputs[1].Value(),
				Database: m.inputs[2].Value(),
				User:     m.inputs[3].Value(),
				Password: m.inputs[4].Value(),
			}

			if err := m.credStore.Save(creds); err != nil {
				m.err = err
				return m, nil
			}

			return m.handleCredentials(creds)

		case "esc":
			m.state = stateExplorer
			return m, nil

		case "tab", "shift+tab", "up", "down":
			s := msg.String()

			if s == "tab" || s == "down" {
				m.inputs[m.activeInput].Blur()
				m.activeInput = (m.activeInput + 1) % len(m.inputs)
				m.inputs[m.activeInput].Focus()
				return m, nil
			}

			if s == "shift+tab" || s == "up" {
				m.inputs[m.activeInput].Blur()
				m.activeInput--
				if m.activeInput < 0 {
					m.activeInput = len(m.inputs) - 1
				}
				m.inputs[m.activeInput].Focus()
				return m, nil
			}
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m model) updateExplorer(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "left", "h":
			m.collapse()
		case "right", "l", "enter":
			m.expand()
		case " ":
			m.toggleSelection()
		case "m":
			md := markdown.Generate(m.schemas)
			if err := clipboard.WriteAll(md); err != nil {
				m.err = err
				return m, nil
			}
			m.message = "Markdown copied to clipboard!"
		case "c":
			if m.cursor.table != -1 {
				m.state = stateComment
				if m.cursor.column != -1 {
					m.commentInput.SetValue(m.schemas[m.cursor.schema].Tables[m.cursor.table].Columns[m.cursor.column].Description)
				} else {
					m.commentInput.SetValue(m.schemas[m.cursor.schema].Tables[m.cursor.table].Description)
				}
				m.commentInput.Focus()
			}
		case "d":
			m.deselectAll()
			m.message = "All items deselected!"
		case "e":
			m.state = stateEditCredentials
			m.message = "Editing connection details..."
		}
	}

	return m, nil
}

func (m *model) getVisibleItems() []cursorPosition {
	var items []cursorPosition

	for s := range m.schemas {
		items = append(items, cursorPosition{
			itemType: "schema",
			schema:   s,
			table:    -1,
			column:   -1,
		})

		if !m.schemas[s].Expanded {
			continue
		}

		for t := range m.schemas[s].Tables {
			items = append(items, cursorPosition{
				itemType: "table",
				schema:   s,
				table:    t,
				column:   -1,
			})

			if !m.schemas[s].Tables[t].Expanded {
				continue
			}

			for c := range m.schemas[s].Tables[t].Columns {
				items = append(items, cursorPosition{
					itemType: "column",
					schema:   s,
					table:    t,
					column:   c,
				})
			}
		}
	}

	return items
}

func (m *model) moveCursor(delta int) {
	items := m.getVisibleItems()
	if len(items) == 0 {
		return
	}

	currentIdx := -1
	for i, item := range items {
		if item.schema == m.cursor.schema &&
			item.table == m.cursor.table &&
			item.column == m.cursor.column {
			currentIdx = i
			break
		}
	}

	newIdx := currentIdx + delta
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx >= len(items) {
		newIdx = len(items) - 1
	}

	newPos := items[newIdx]
	m.cursor.schema = newPos.schema
	m.cursor.table = newPos.table
	m.cursor.column = newPos.column
}

func (m *model) expand() {
	if m.cursor.schema < 0 || m.cursor.schema >= len(m.schemas) {
		return
	}

	schema := &m.schemas[m.cursor.schema]
	if m.cursor.table == -1 {
		schema.Expanded = true
		if len(schema.Tables) > 0 {
			m.cursor.table = 0
		}
		return
	}

	if m.cursor.table >= len(schema.Tables) {
		return
	}

	table := &schema.Tables[m.cursor.table]
	if m.cursor.column == -1 {
		table.Expanded = true
		if len(table.Columns) > 0 {
			m.cursor.column = 0
		}
	}
}

func (m *model) collapse() {
	if m.cursor.schema < 0 || m.cursor.schema >= len(m.schemas) {
		return
	}

	schema := &m.schemas[m.cursor.schema]
	if m.cursor.column != -1 {
		m.cursor.column = -1
		return
	}

	if m.cursor.table != -1 {
		if m.cursor.table >= len(schema.Tables) {
			return
		}
		table := &schema.Tables[m.cursor.table]
		if table.Expanded {
			table.Expanded = false
		} else {
			m.cursor.table = -1
		}
		return
	}

	if schema.Expanded {
		schema.Expanded = false
		m.cursor.table = -1
		m.cursor.column = -1
	}
}

func (m *model) toggleSelection() {
	if m.cursor.schema < 0 || m.cursor.schema >= len(m.schemas) {
		return
	}

	schema := &m.schemas[m.cursor.schema]

	if m.cursor.table == -1 {
		// Toggle schema selection
		schema.Selected = !schema.Selected
		for i := range schema.Tables {
			table := &schema.Tables[i]
			table.Selected = schema.Selected
			for j := range table.Columns {
				table.Columns[j].Selected = schema.Selected
			}
		}
		return
	}

	if m.cursor.table >= len(schema.Tables) {
		return
	}

	table := &schema.Tables[m.cursor.table]
	if m.cursor.column == -1 {
		// Toggle table selection
		table.Selected = !table.Selected
		for i := range table.Columns {
			table.Columns[i].Selected = table.Selected
		}

		// Update schema selection
		allSelected := true
		for _, t := range schema.Tables {
			if !t.Selected {
				allSelected = false
				break
			}
		}
		schema.Selected = allSelected
		return
	}

	if m.cursor.column >= len(table.Columns) {
		return
	}

	// Toggle column selection
	column := &table.Columns[m.cursor.column]
	column.Selected = !column.Selected

	// Update table selection
	allSelected := true
	for _, col := range table.Columns {
		if !col.Selected {
			allSelected = false
			break
		}
	}
	table.Selected = allSelected

	// Update schema selection
	allTablesSelected := true
	for _, t := range schema.Tables {
		if !t.Selected {
			allTablesSelected = false
			break
		}
	}
	schema.Selected = allTablesSelected
}

func (m model) updateComment(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Save the comment
			commentText := m.commentInput.Value()
			var schema, table, column string
			schema = m.schemas[m.cursor.schema].Name

			if m.cursor.column != -1 {
				table = m.schemas[m.cursor.schema].Tables[m.cursor.table].Name
				column = m.schemas[m.cursor.schema].Tables[m.cursor.table].Columns[m.cursor.column].Name
			} else if m.cursor.table != -1 {
				table = m.schemas[m.cursor.schema].Tables[m.cursor.table].Name
			} else {
				m.err = fmt.Errorf("no table or column selected")
				return m, nil
			}

			ctx := context.Background()

			// Update the comment
			err := m.client.UpdateComment(ctx, schema, table, column, commentText)
			if err != nil {
				switch {
				case errors.Is(err, postgres.ErrCommentTooLong):
					m.err = fmt.Errorf("comment is too long (max 1000 characters)")
				case errors.Is(err, postgres.ErrCommentEmpty):
					m.err = fmt.Errorf("comment cannot be empty")
				case errors.Is(err, postgres.ErrCommentMalicious):
					m.err = fmt.Errorf("comment contains invalid characters or patterns")
				default:
					m.err = err
				}
				return m, nil
			}

			// Verify the comment was stored
			storedComment, err := m.client.VerifyComment(ctx, schema, table, column)
			if err != nil {
				m.err = fmt.Errorf("failed to verify comment: %w", err)
				return m, nil
			}

			if storedComment != commentText {
				m.err = fmt.Errorf("comment verification failed: expected %q, got %q", commentText, storedComment)
				return m, nil
			}

			// Update the comment in the model
			if m.cursor.column != -1 {
				m.schemas[m.cursor.schema].Tables[m.cursor.table].Columns[m.cursor.column].Description = commentText
			} else {
				m.schemas[m.cursor.schema].Tables[m.cursor.table].Description = commentText
			}

			// Return to explorer state
			m.state = stateExplorer
			m.commentInput.Reset()
			m.message = "Comment updated and verified successfully!"
			return m, nil

		case tea.KeyEsc:
			// Cancel and return to explorer state
			m.state = stateExplorer
			m.commentInput.Reset()
			m.err = nil
			m.message = "Comment update cancelled"
			return m, nil
		}

		// Handle all other key events with the textinput component
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}
