// internal/ui/views.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginLeft(0).
			PaddingLeft(0)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginLeft(0).
			PaddingLeft(0)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			MarginLeft(0).
			PaddingLeft(0)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			MarginLeft(0).
			PaddingLeft(0)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			MarginLeft(0).
			PaddingLeft(0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(0).
			PaddingLeft(0)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("111")).
			MarginLeft(0).
			PaddingLeft(0)
)

func (m model) credentialsView() string {
	var b strings.Builder

	// Remove the newlines from the title render and handle spacing explicitly
	b.WriteString(titleStyle.Render("PostgreSQL Connection Details"))
	b.WriteString("\n\n")

	labels := []string{"Host:", "Port:", "Database:", "User:", "Password:"}
	for i := range m.inputs {
		label := inputLabelStyle.Render(labels[i])
		inputView := m.inputs[i].View()
		line := fmt.Sprintf("%-12s %s", label, inputView)
		b.WriteString(line + "\n")
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(wordwrap.String(m.err.Error(), m.width)))
	}

	help := "\nPress Enter to connect, Esc to cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) commentView() string {
	var b strings.Builder

	// Get current item
	var itemType, itemName, currentComment string
	if m.cursor.column != -1 {
		schema := m.schemas[m.cursor.schema]
		table := schema.Tables[m.cursor.table]
		column := table.Columns[m.cursor.column]
		itemType = "column"
		itemName = fmt.Sprintf("%s.%s.%s", schema.Name, table.Name, column.Name)
		currentComment = column.Description
	} else if m.cursor.table != -1 {
		schema := m.schemas[m.cursor.schema]
		table := schema.Tables[m.cursor.table]
		itemType = "table"
		itemName = fmt.Sprintf("%s.%s", schema.Name, table.Name)
		currentComment = table.Description
	} else {
		return "Please select a table or column to comment"
	}

	// Title without newlines in the style render
	b.WriteString(titleStyle.Render(fmt.Sprintf("Adding comment to %s: %s", itemType, itemName)))
	b.WriteString("\n\n")

	// Current comment
	if currentComment != "" {
		b.WriteString("Current comment: ")
		b.WriteString(infoStyle.Render(currentComment))
		b.WriteString("\n\n")
	}

	// New comment input
	newCommentLabel := "New comment:"
	line := fmt.Sprintf("%-15s %s", newCommentLabel, m.commentInput.View())
	b.WriteString(line + "\n\n")

	// Help text
	help := "Press Enter to save, Esc to cancel"
	b.WriteString(helpStyle.Render(help))

	// Error message
	if m.err != nil {
		b.WriteString("\n\n" + errorStyle.Render(wordwrap.String(m.err.Error(), m.width)))
	}

	return b.String()
}

func (m model) explorerView() string {
	var b strings.Builder

	// Help text at the top
	help := "↑/↓: navigate • space: select • →/←: expand/collapse • d: deselect all • e: edit connection details • m: markdown • c: comment • q: quit\n"
	b.WriteString(helpStyle.Render(wordwrap.String(help, m.width)))
	b.WriteString("\n")

	// Render schemas
	for i, schema := range m.schemas {
		style := normalStyle
		if m.cursor.schema == i && m.cursor.table == -1 && m.cursor.column == -1 {
			style = selectedStyle
		}

		// Schema line
		schemaMarker := "▼"
		if !schema.Expanded {
			schemaMarker = "▶"
		}

		schemaLine := schemaMarker + " " + schema.Name
		if schema.Selected {
			schemaLine = "* " + schemaLine
		}

		b.WriteString(style.Render(wordwrap.String(schemaLine, m.width)) + "\n")

		if schema.Expanded {
			// Render tables
			for j, table := range schema.Tables {
				style := normalStyle
				if m.cursor.schema == i && m.cursor.table == j && m.cursor.column == -1 {
					style = selectedStyle
				}

				// Construct table line
				indent := "    "
				marker := "▼"
				if !table.Expanded {
					marker = "▶"
				}

				tableLine := fmt.Sprintf("%s%s %s", indent, marker, table.Name)
				if table.Selected {
					tableLine = fmt.Sprintf("%s* %s", indent, marker+" "+table.Name)
				}

				if table.Description != "" {
					description := wordwrap.String(table.Description, m.width-len(indent)-4)
					descriptionLines := strings.Split(description, "\n")
					for i, line := range descriptionLines {
						if i == 0 {
							tableLine += " " + infoStyle.Render("- "+line)
						} else {
							tableLine += "\n" + infoStyle.Render(strings.Repeat(" ", len(indent)+4)+line)
						}
					}
				}

				b.WriteString(style.Render(wordwrap.String(tableLine, m.width)) + "\n")

				if table.Expanded {
					// Render columns
					for k, col := range table.Columns {
						style := normalStyle
						if m.cursor.schema == i && m.cursor.table == j && m.cursor.column == k {
							style = selectedStyle
						}

						columnIndent := "        "
						columnPrefix := "  "
						if col.Selected {
							columnPrefix = "* "
						}

						columnLine := columnIndent + columnPrefix + col.Name
						if col.Type != "" {
							columnLine += ": " + col.Type

							constraints := []string{}
							if !col.IsNullable {
								constraints = append(constraints, "NOT NULL")
							}
							if col.IsPrimary {
								constraints = append(constraints, "PRIMARY KEY")
							}
							if col.IsUnique {
								constraints = append(constraints, "UNIQUE")
							}
							if col.HasDefault {
								constraints = append(constraints, fmt.Sprintf("DEFAULT %s", col.Default))
							}
							if len(col.Constraints) > 0 {
								constraints = append(constraints, col.Constraints...)
							}

							if len(constraints) > 0 {
								columnLine += " (" + strings.Join(constraints, ", ") + ")"
							}
						}

						if col.Description != "" {
							description := wordwrap.String(col.Description, m.width-len(columnIndent)-4)
							descriptionLines := strings.Split(description, "\n")
							for i, line := range descriptionLines {
								if i == 0 {
									columnLine += " " + infoStyle.Render("- "+line)
								} else {
									columnLine += "\n" + infoStyle.Render(strings.Repeat(" ", len(columnIndent)+4)+line)
								}
							}
						}

						b.WriteString(style.Render(wordwrap.String(columnLine, m.width)) + "\n")
					}
				}
			}
		}
	}

	if m.message != "" {
		b.WriteString("\n" + infoStyle.Render(wordwrap.String(m.message, m.width)))
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(wordwrap.String(m.err.Error(), m.width)))
	}

	return b.String()
}
