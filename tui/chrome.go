package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/rivo/tview"
)

const appVersion = "v0.1.0"

type keyHint struct {
	key    string
	action string
}

// makeHeader returns a single-line top bar with branding, breadcrumb and root indicator.
func makeHeader(breadcrumb string) *tview.TextView {
	tv := tview.NewTextView().SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignLeft)

	rootBadge := fmt.Sprintf("[%s]user[-]", cMuted)
	if os.Geteuid() == 0 {
		rootBadge = fmt.Sprintf("[%s::b]root[-:-:-]", cWarning)
	}

	tv.SetText(fmt.Sprintf(
		" [%s::b]aether[-:-:-] [%s]›[-] [%s]%s[-]   [%s]%s · %s[-]",
		cBrand, cMuted, cText, tview.Escape(breadcrumb),
		cMuted, rootBadge, appVersion,
	))
	return tv
}

// makeFooter returns a single-line bottom keybind bar.
func makeFooter(hints []keyHint) *tview.TextView {
	tv := tview.NewTextView().SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignLeft)

	var b strings.Builder
	b.WriteString(" ")
	for i, h := range hints {
		if i > 0 {
			b.WriteString(fmt.Sprintf("  [%s]·[-]  ", cBorder))
		}
		b.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-] [%s]%s[-]", cKey, h.key, cSubtext, h.action))
	}
	tv.SetText(b.String())
	return tv
}

// chrome wraps a body primitive with header and footer rows. The body keeps focus.
func chrome(body tview.Primitive, breadcrumb string, hints []keyHint) tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(makeHeader(breadcrumb), 1, 0, false).
		AddItem(spacer(), 1, 0, false).
		AddItem(body, 0, 1, true).
		AddItem(spacer(), 1, 0, false).
		AddItem(makeFooter(hints), 1, 0, false)
}

// spacer returns a blank box used as vertical breathing room between header/body/footer.
func spacer() *tview.Box {
	return tview.NewBox()
}

// commonBackHint is the universal back hint used in body footers.
var commonBackHint = keyHint{"esc", "back"}
