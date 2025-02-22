package markdown

import (
	"fmt"
	"strings"
	"time"

	"github.com/kerem-kaynak/llmshark/internal/postgres"
)

func Generate(schemas []postgres.Schema) string {
	var b strings.Builder

	b.WriteString("# Database Schema Documentation\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Iterate through schemas
	for _, schema := range schemas {
		// Check if the schema or any of its tables/columns are selected
		schemaHasSelection := schema.Selected
		for _, table := range schema.Tables {
			if table.Selected {
				schemaHasSelection = true
				break
			}
			for _, col := range table.Columns {
				if col.Selected {
					schemaHasSelection = true
					break
				}
			}
		}

		// Skip schema if nothing is selected
		if !schemaHasSelection {
			continue
		}

		// Add schema header
		b.WriteString(fmt.Sprintf("## Schema: `%s`\n\n", schema.Name))

		// Iterate through tables
		for _, table := range schema.Tables {
			// Check if the table or any of its columns are selected
			tableHasSelection := table.Selected || schema.Selected
			if !tableHasSelection {
				for _, col := range table.Columns {
					if col.Selected {
						tableHasSelection = true
						break
					}
				}
			}

			// Skip table if nothing is selected
			if !tableHasSelection {
				continue
			}

			// Add table header
			b.WriteString(fmt.Sprintf("### Table: `%s`\n\n", table.Name))
			if table.Description != "" {
				b.WriteString(fmt.Sprintf("%s\n\n", table.Description))
			}

			// Add columns header
			b.WriteString("#### Columns\n\n")
			b.WriteString("| Name | Type | Constraints | Description |\n")
			b.WriteString("|------|------|-------------|-------------|\n")

			// Iterate through columns
			for _, col := range table.Columns {
				// Include column if it's selected, or if the table/schema is selected
				if !col.Selected && !table.Selected && !schema.Selected {
					continue
				}

				// Build constraints
				constraints := make([]string, 0)
				if col.IsPrimary {
					constraints = append(constraints, "PRIMARY KEY")
				}
				if col.IsUnique {
					constraints = append(constraints, "UNIQUE")
				}
				if !col.IsNullable {
					constraints = append(constraints, "NOT NULL")
				}
				if col.HasDefault {
					constraints = append(constraints, fmt.Sprintf("DEFAULT %s", col.Default))
				}
				if len(col.Constraints) > 0 {
					constraints = append(constraints, col.Constraints...)
				}

				constraintStr := "-"
				if len(constraints) > 0 {
					constraintStr = strings.Join(constraints, ", ")
				}

				desc := col.Description
				if desc == "" {
					desc = "-"
				}

				// Add column row
				fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n",
					col.Name,
					col.Type,
					strings.ReplaceAll(constraintStr, "|", "\\|"),
					strings.ReplaceAll(desc, "|", "\\|"))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}
