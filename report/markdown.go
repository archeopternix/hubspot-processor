package report

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/archeopternix/hubspot-processor/domain"
)

// MarkdownPrinter renders CRM objects as a human-readable Markdown report.
type MarkdownPrinter struct {
	definition domain.ObjectDefinition
}

func NewMarkdownPrinter(definition domain.ObjectDefinition) *MarkdownPrinter {
	return &MarkdownPrinter{definition: definition}
}

func (p *MarkdownPrinter) PrintOne(writer io.Writer, object *domain.Object) error {
	if writer == nil {
		return fmt.Errorf("Markdown print: writer is nil")
	}
	if object == nil {
		return fmt.Errorf("Markdown print: object is nil")
	}

	var b strings.Builder
	p.writeObject(&b, object)

	if _, err := io.WriteString(writer, b.String()); err != nil {
		return fmt.Errorf("Markdown print: write output: %w", err)
	}
	return nil
}

func (p *MarkdownPrinter) PrintAll(writer io.Writer, objects []domain.Object) error {
	if writer == nil {
		return fmt.Errorf("Markdown print: writer is nil")
	}

	if _, err := fmt.Fprintf(writer, "# HubSpot %s Result\n\n", objectTypeLabel(p.definition.Type)); err != nil {
		return fmt.Errorf("Markdown print: write output: %w", err)
	}

	for i := range objects {
		if i > 0 {
			if _, err := io.WriteString(writer, "\n---\n\n"); err != nil {
				return fmt.Errorf("Markdown print: write output: %w", err)
			}
		}
		if err := p.PrintOne(writer, &objects[i]); err != nil {
			return err
		}
	}

	return nil
}

func (p *MarkdownPrinter) writeObject(b *strings.Builder, object *domain.Object) {
	fmt.Fprintf(b, "**ID:** %s\n\n", escapeMarkdownText(object.ID))

	for _, attribute := range p.definition.Attributes {
		if attribute.Export {
			continue
		}
		fmt.Fprintf(
			b,
			"**%s:** %s\n",
			escapeMarkdownText(attribute.Name),
			escapeMarkdownText(truncate(object.ImportedValue(attribute.Name), 25)),
		)
	}

	b.WriteString("\n")
	b.WriteString("| Attribute | Import | Proposal | Score | Export |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, attribute := range p.definition.Attributes {
		if !attribute.Export {
			continue
		}
		state := object.Attributes[attribute.Name]
		fmt.Fprintf(
			b,
			"| %s | %s | %s | %s | %s |\n",
			escapeMarkdownCell(attribute.Name),
			escapeMarkdownCell(truncate(state.Import, 25)),
			escapeMarkdownCell(truncate(state.Proposal, 25)),
			renderScore(state.Score, attribute.Research),
			escapeMarkdownCell(truncate(state.Export, 25)),
		)
	}

	for _, attribute := range p.definition.Attributes {
		state := object.Attributes[attribute.Name]
		if len(state.Proposal) <= 25 {
			continue
		}

		fmt.Fprintf(b, "\n**%s Proposal:**\n\n", escapeMarkdownText(attribute.Name))
		b.WriteString(strings.TrimSpace(normalizeLineEndings(state.Proposal)))
		b.WriteString("\n")
	}
}

func objectTypeLabel(objectType string) string {
	switch strings.ToLower(strings.TrimSpace(objectType)) {
	case "companies":
		return "Company"
	case "contacts":
		return "Contact"
	default:
		return "Object"
	}
}

func renderScore(score float64, researched bool) string {
	if !researched {
		return ""
	}
	return fmt.Sprintf("%.2f", score)
}

func truncate(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	if maxChars <= 0 || value == "" {
		return value
	}
	if utf8.RuneCountInString(value) <= maxChars {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxChars]) + "..."
}

func escapeMarkdownText(value string) string {
	value = normalizeLineEndings(value)
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = normalizeLineEndings(value)
	value = strings.ReplaceAll(value, "\n", "<br>")
	return strings.TrimSpace(value)
}

func normalizeLineEndings(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}
