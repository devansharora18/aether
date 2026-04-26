package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	cAccent      = "#89b4fa"
	cTitle       = "#cba6f7"
	cBorder      = "#45475a"
	cBorderFocus = "#89b4fa"
	cMuted       = "#6c7086"
	cText        = "#cdd6f4"
	cSubtext     = "#a6adc8"
	cKey         = "#f9e2af"
	cSuccess     = "#a6e3a1"
	cWarning     = "#fab387"
	cError       = "#f38ba8"
	cInfo        = "#89dceb"
	cBrand       = "#cba6f7"
)

var (
	colAccent      = tcell.GetColor(cAccent)
	colBorder      = tcell.GetColor(cBorder)
	colBorderFocus = tcell.GetColor(cBorderFocus)
	colTitle       = tcell.GetColor(cTitle)
	colMuted       = tcell.GetColor(cMuted)
	colText        = tcell.GetColor(cText)
	colSubtext     = tcell.GetColor(cSubtext)
	colKey         = tcell.GetColor(cKey)
	colSuccess     = tcell.GetColor(cSuccess)
	colError       = tcell.GetColor(cError)
)

func setupTheme() {
	tview.Borders.Horizontal = '─'
	tview.Borders.Vertical = '│'
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'
	tview.Borders.LeftT = '├'
	tview.Borders.RightT = '┤'
	tview.Borders.TopT = '┬'
	tview.Borders.BottomT = '┴'
	tview.Borders.Cross = '┼'
	tview.Borders.HorizontalFocus = '─'
	tview.Borders.VerticalFocus = '│'
	tview.Borders.TopLeftFocus = '╭'
	tview.Borders.TopRightFocus = '╮'
	tview.Borders.BottomLeftFocus = '╰'
	tview.Borders.BottomRightFocus = '╯'

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = colAccent
	tview.Styles.MoreContrastBackgroundColor = colTitle
	tview.Styles.BorderColor = colBorder
	tview.Styles.TitleColor = colTitle
	tview.Styles.GraphicsColor = colBorder
	tview.Styles.PrimaryTextColor = colText
	tview.Styles.SecondaryTextColor = colSubtext
	tview.Styles.TertiaryTextColor = colTitle
	tview.Styles.InverseTextColor = colAccent
	tview.Styles.ContrastSecondaryTextColor = colText
}

// styleList applies the standard list look (selection highlight, muted secondary text).
func styleList(l *tview.List) *tview.List {
	l.SetMainTextColor(colText).
		SetSecondaryTextColor(colMuted).
		SetShortcutColor(colKey).
		SetSelectedTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(colAccent).
		SetHighlightFullLine(true)
	l.SetBorderColor(colBorder).SetTitleColor(colTitle)
	return l
}

// stylePanel applies the standard bordered-panel look to anything that embeds Box.
func stylePanel(b *tview.Box) *tview.Box {
	b.SetBorderColor(colBorder).
		SetTitleColor(colTitle).
		SetTitleAlign(tview.AlignLeft)
	return b
}

// styleForm applies the standard form look (label + field colors, button colors).
func styleForm(f *tview.Form) *tview.Form {
	f.SetLabelColor(colSubtext).
		SetFieldTextColor(colText).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetButtonTextColor(tcell.ColorBlack).
		SetButtonBackgroundColor(colAccent).
		SetButtonActivatedStyle(tcell.StyleDefault.Background(colTitle).Foreground(tcell.ColorBlack))
	f.SetBorderColor(colBorder).SetTitleColor(colTitle).SetTitleAlign(tview.AlignLeft)
	return f
}

// styleModal applies a consistent modal look.
func styleModal(m *tview.Modal) *tview.Modal {
	m.SetBackgroundColor(tcell.ColorDefault).
		SetTextColor(colText).
		SetButtonTextColor(tcell.ColorBlack).
		SetButtonBackgroundColor(colAccent).
		SetButtonActivatedStyle(tcell.StyleDefault.Background(colTitle).Foreground(tcell.ColorBlack))
	m.SetBorderColor(colBorder).SetTitleColor(colTitle).SetTitleAlign(tview.AlignLeft)
	return m
}
