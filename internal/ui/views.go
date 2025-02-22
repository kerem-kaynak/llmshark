// internal/ui/views.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("111"))
)

func (m model) credentialsView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("PostgreSQL Connection Details\n\n"))

	labels := []string{"Host:", "Port:", "Database:", "User:", "Password:"}
	for i := range m.inputs {
		label := inputLabelStyle.Render(fmt.Sprintf("%-10s ", labels[i]))
		b.WriteString(label)
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(m.err.Error()))
	}

	help := "\nPress Enter to connect, Esc to cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) explorerView() string {
	var b strings.Builder

	// Help text at the top
	help := "↑/↓: navigate • space: select • →/←: expand/collapse • m: markdown • c: comment • q: quit\n"
	b.WriteString(helpStyle.Render(help))
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

		b.WriteString(style.Render(schemaLine) + "\n")

		if schema.Expanded {
			// Render tables
			for j, table := range schema.Tables {
				style := normalStyle
				if m.cursor.schema == i && m.cursor.table == j && m.cursor.column == -1 {
					style = selectedStyle
				}

				// Construct table line with explicit indentation
				indent := "    " // Exactly 4 spaces
				marker := "▼"
				if !table.Expanded {
					marker = "▶"
				}

				tableLine := fmt.Sprintf("%s%s %s", indent, marker, table.Name)

				if table.Selected {
					tableLine = fmt.Sprintf("%s* %s", indent, marker+" "+table.Name)
				}

				b.WriteString(style.Render(tableLine))
				if table.Description != "" {
					b.WriteString(infoStyle.Render(" - " + table.Description))
				}
				b.WriteString("\n")

				if table.Expanded {
					// Render columns
					for k, col := range table.Columns {
						style := normalStyle
						if m.cursor.schema == i && m.cursor.table == j && m.cursor.column == k {
							style = selectedStyle
						}

						// Column line with explicit indentation
						columnIndent := "        " // Exactly 8 spaces
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

						b.WriteString(style.Render(columnLine))
						if col.Description != "" {
							b.WriteString(infoStyle.Render(" - " + col.Description))
						}
						b.WriteString("\n")
					}
				}
			}
		}
	}

	if m.message != "" {
		b.WriteString("\n" + infoStyle.Render(m.message))
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(m.err.Error()))
	}

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

	b.WriteString(titleStyle.Render(fmt.Sprintf("Adding comment to %s: %s\n\n", itemType, itemName)))

	if currentComment != "" {
		b.WriteString("Current comment: ")
		b.WriteString(infoStyle.Render(currentComment))
		b.WriteString("\n\n")
	}

	b.WriteString("New comment: ")
	b.WriteString(m.commentText)

	help := "\n\nPress Enter to save, Esc to cancel"
	b.WriteString(helpStyle.Render(help))

	if m.err != nil {
		b.WriteString("\n\n" + errorStyle.Render(m.err.Error()))
	}

	return b.String()
}
