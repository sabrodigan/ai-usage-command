package ui

import (
	"fmt"
	"strings"
)

// VisualWidth returns the displayed column width of a string in a monospace terminal.
func VisualWidth(s string) int {
	w := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		// Handle Zero-width characters
		if (r >= 0xFE00 && r <= 0xFE0F) || r == 0x200D || r == 0x200E || r == 0x200F {
			continue
		}

		// Handle Single-width symbols and arrows in Dingbats / Miscellaneous ranges
		if r == '➔' || r == '✔' || r == '✓' || r == 0x2699 || (r >= 0x2790 && r <= 0x27AF) {
			w += 1
			continue
		}

		// Handle Double-width Emojis and CJK characters
		if (r >= 0x1F300 && r <= 0x1FAFF) || // Emojis & Pictographs
			(r >= 0x2600 && r <= 0x27BF) ||   // Miscellaneous Symbols & Dingbats
			(r >= 0x2300 && r <= 0x23FF) ||   // Misc Technical
			(r >= 0x4E00 && r <= 0x9FFF) ||   // CJK Unified Ideographs
			(r >= 0x3000 && r <= 0x303F) ||   // CJK Symbols and Punctuation
			(r >= 0xFF01 && r <= 0xFF60) {    // Fullwidth Forms
			w += 2
		} else {
			w += 1
		}
	}
	return w
}

// FormatBoxTop renders a top border: ┌───...───┐
func FormatBoxTop(width int) string {
	if width < 2 {
		width = 90
	}
	return "┌" + strings.Repeat("─", width-2) + "┐\n"
}

// FormatBoxDivider renders a middle separator: ├───...───┤
func FormatBoxDivider(width int) string {
	if width < 2 {
		width = 90
	}
	return "├" + strings.Repeat("─", width-2) + "┤\n"
}

// FormatBoxBottom renders a bottom border: └───...───┘
func FormatBoxBottom(width int) string {
	if width < 2 {
		width = 90
	}
	return "└" + strings.Repeat("─", width-2) + "┘\n"
}

// FormatBoxLine formats content within left '│  ' and right '  │' borders with exact padding.
func FormatBoxLine(content string, boxWidth int) string {
	if boxWidth < 10 {
		boxWidth = 90
	}
	// interior space for content between '│  ' (3 chars) and '  │' (3 chars)
	innerWidth := boxWidth - 6
	if innerWidth < 1 {
		innerWidth = 1
	}

	vw := VisualWidth(content)
	pad := innerWidth - vw
	if pad < 0 {
		pad = 0
	}

	return fmt.Sprintf("│  %s%s  │\n", content, strings.Repeat(" ", pad))
}

// FormatTwoColumnBoxLine formats two columns aligned within a single box row.
func FormatTwoColumnBoxLine(left string, leftWidth int, right string, boxWidth int) string {
	if boxWidth < 10 {
		boxWidth = 90
	}
	innerWidth := boxWidth - 6
	leftVw := VisualWidth(left)
	leftPad := leftWidth - leftVw
	if leftPad < 0 {
		leftPad = 0
	}
	leftPadded := left + strings.Repeat(" ", leftPad)

	combined := leftPadded + "  " + right
	combVw := VisualWidth(combined)
	pad := innerWidth - combVw
	if pad < 0 {
		pad = 0
	}

	return fmt.Sprintf("│  %s%s  │\n", combined, strings.Repeat(" ", pad))
}
