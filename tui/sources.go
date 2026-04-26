package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/devansharora18/aether/libapt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func showSources(app *tview.Application, pages *tview.Pages) {
	sourcesList := styleList(tview.NewList().ShowSecondaryText(true))
	sourcesList.SetBorder(true).SetTitle(" APT Sources ")
	stylePanel(sourcesList.Box)

	preview := tview.NewTextView().SetDynamicColors(true)
	preview.SetBorder(true).SetTitle(" Details ")
	stylePanel(preview.Box)

	var currentEntries []libapt.SourceEntry

	loadSources := func() {
		sourcesList.Clear()
		currentEntries = nil
		preview.SetText(fmt.Sprintf("\n  [%s]Loading sources…[-]", cMuted))

		entries, err := libapt.ListSources()
		if err != nil {
			preview.SetText(fmt.Sprintf("\n  [%s::b]✗ Failed to load sources[-:-:-]\n  %v", cError, err))
			return
		}
		currentEntries = entries

		if len(entries) == 0 {
			sourcesList.AddItem("(no sources found)", "", 0, nil)
			preview.SetText(fmt.Sprintf("\n  [%s]No APT source entries found on this system.[-]", cMuted))
			return
		}

		for _, e := range entries {
			marker := fmt.Sprintf("[%s]●[-]", cSuccess)
			if !e.Enabled {
				marker = fmt.Sprintf("[%s]○[-]", cMuted)
			}
			primary := fmt.Sprintf("%s  %s", marker, e.DisplayString())
			secondary := fmt.Sprintf("[%s]%s[-]", cSubtext, filepath.Base(e.FilePath))
			sourcesList.AddItem(primary, secondary, 0, nil)
		}

		sourcesList.SetTitle(fmt.Sprintf(" APT Sources · %d ", len(entries)))
		if len(entries) > 0 {
			sourcesList.SetCurrentItem(0)
		}
	}

	renderPreview := func(idx int) {
		if idx < 0 || idx >= len(currentEntries) {
			preview.SetText(fmt.Sprintf("\n  [%s]No source selected[-]", cMuted))
			return
		}
		e := currentEntries[idx]
		var b strings.Builder
		b.WriteString("\n")

		if e.Enabled {
			b.WriteString(fmt.Sprintf("  [%s::b]● Enabled[-:-:-]\n\n", cSuccess))
		} else {
			b.WriteString(fmt.Sprintf("  [%s::b]○ Disabled[-:-:-]\n\n", cError))
		}

		row := func(k, v string) {
			if v == "" {
				return
			}
			b.WriteString(fmt.Sprintf("  [%s]%-12s[-] %s\n", cSubtext, k, v))
		}
		row("Type", e.Type)
		row("URI", e.URI)
		row("Suite", e.Suite)
		if len(e.Components) > 0 {
			row("Components", strings.Join(e.Components, " "))
		}
		if e.Options != "" {
			row("Options", "["+e.Options+"]")
		}
		b.WriteString("\n")
		row("File", e.FilePath)
		if e.LineNumber > 0 {
			row("Line", fmt.Sprintf("%d", e.LineNumber))
		}
		if e.IsDEB822 {
			row("Format", "DEB822 (.sources)")
		} else {
			row("Format", "One-line (.list)")
		}

		b.WriteString(fmt.Sprintf("\n  [%s]Raw[-]\n  [%s]%s[-]\n", cSubtext, cMuted, e.RawLine))
		if e.IsDEB822 && e.DEB822Block != "" {
			b.WriteString(fmt.Sprintf("\n  [%s]Stanza[-]\n", cSubtext))
			for _, line := range strings.Split(e.DEB822Block, "\n") {
				b.WriteString(fmt.Sprintf("  [%s]%s[-]\n", cMuted, line))
			}
		}

		preview.SetText(b.String())
	}

	sourcesList.SetChangedFunc(func(index int, _, _ string, _ rune) {
		renderPreview(index)
	})

	showActionPopup := func(idx int) {
		if idx < 0 || idx >= len(currentEntries) {
			return
		}
		entry := currentEntries[idx]

		toggleLabel := "Disable"
		if !entry.Enabled {
			toggleLabel = "Enable"
		}

		modal := tview.NewModal().
			SetText(fmt.Sprintf("Source: %s\n%s %s", entry.URI, entry.Suite, strings.Join(entry.Components, " "))).
			AddButtons([]string{toggleLabel, "Edit", "Delete", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				pages.RemovePage("source-action-modal")

				switch buttonLabel {
				case "Enable", "Disable":
					if !ensureRoot(app, pages) {
						return
					}
					err := libapt.ToggleSource(entry)
					if err != nil {
						showInfoModal(app, pages, "Error", fmt.Sprintf("Failed to toggle source:\n%v", err))
						return
					}
					loadSources()
					renderPreview(sourcesList.GetCurrentItem())

				case "Edit":
					if !ensureRoot(app, pages) {
						return
					}
					showEditSourceForm(app, pages, &entry, func() {
						loadSources()
						renderPreview(sourcesList.GetCurrentItem())
					})

				case "Delete":
					if !ensureRoot(app, pages) {
						return
					}
					showDeleteConfirm(app, pages, entry, func() {
						loadSources()
						if sourcesList.GetItemCount() > 0 {
							renderPreview(sourcesList.GetCurrentItem())
						} else {
							preview.SetText(fmt.Sprintf("\n  [%s]No sources[-]", cMuted))
						}
					})
				}
			})
		styleModal(modal)
		modal.SetTitle(" Source Actions ").SetBorder(true)
		pages.AddPage("source-action-modal", modal, true, true)
	}

	sourcesList.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		showActionPopup(index)
	})

	body := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(sourcesList, 0, 1, true).
		AddItem(preview, 0, 1, false)

	body.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			pages.SwitchToPage("menu")
			pages.RemovePage("sources")
			return nil
		}

		switch event.Rune() {
		case 'a':
			if !ensureRoot(app, pages) {
				return nil
			}
			showAddSourceForm(app, pages, func() {
				loadSources()
				if sourcesList.GetItemCount() > 0 {
					sourcesList.SetCurrentItem(sourcesList.GetItemCount() - 1)
					renderPreview(sourcesList.GetCurrentItem())
				}
			})
			return nil

		case 'e':
			showFileEditor(app, pages, func() {
				loadSources()
				renderPreview(sourcesList.GetCurrentItem())
			})
			return nil
		}

		return event
	})

	hints := []keyHint{
		commonBackHint,
		{"↵", "actions"},
		{"a", "add"},
		{"e", "edit file"},
	}
	pages.AddAndSwitchToPage("sources", chrome(body, "Manage APT Sources", hints), true)
	loadSources()
	if len(currentEntries) > 0 {
		renderPreview(0)
	}
}

// showAddSourceForm shows a form to add a new APT source entry.
func showAddSourceForm(app *tview.Application, pages *tview.Pages, onDone func()) {
	form := styleForm(tview.NewForm())

	form.AddDropDown("Type", []string{"deb", "deb-src"}, 0, nil)
	form.AddInputField("URI", "http://", 60, nil, nil)
	form.AddInputField("Suite", "", 40, nil, nil)
	form.AddInputField("Components", "main", 40, nil, nil)
	form.AddInputField("Options", "", 60, nil, nil)
	form.AddInputField("File", "/etc/apt/sources.list.d/aether-custom.list", 60, nil, nil)
	form.AddCheckbox("Enabled", true, nil)

	form.AddButton("Add", func() {
		_, typVal := form.GetFormItemByLabel("Type").(*tview.DropDown).GetCurrentOption()
		uri := strings.TrimSpace(form.GetFormItemByLabel("URI").(*tview.InputField).GetText())
		suite := strings.TrimSpace(form.GetFormItemByLabel("Suite").(*tview.InputField).GetText())
		comps := strings.TrimSpace(form.GetFormItemByLabel("Components").(*tview.InputField).GetText())
		opts := strings.TrimSpace(form.GetFormItemByLabel("Options").(*tview.InputField).GetText())
		filePath := strings.TrimSpace(form.GetFormItemByLabel("File").(*tview.InputField).GetText())
		enabled := form.GetFormItemByLabel("Enabled").(*tview.Checkbox).IsChecked()

		if uri == "" || uri == "http://" || suite == "" {
			showInfoModal(app, pages, "Missing fields", "URI and Suite are required.")
			return
		}

		entry := libapt.SourceEntry{
			Enabled:  enabled,
			Type:     typVal,
			URI:      uri,
			Suite:    suite,
			Options:  opts,
			FilePath: filePath,
		}
		if comps != "" {
			entry.Components = strings.Fields(comps)
		}

		err := libapt.AddSource(entry)
		if err != nil {
			showInfoModal(app, pages, "Error", fmt.Sprintf("Failed to add source:\n%v", err))
			return
		}

		pages.RemovePage("add-source")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("Cancel", func() {
		pages.RemovePage("add-source")
	})

	form.SetBorder(true).SetTitle(" Add New Source ")
	pages.AddPage("add-source", centered(form, 80, 22), true, true)
}

// showEditSourceForm shows a form pre-filled with source entry data for editing.
func showEditSourceForm(app *tview.Application, pages *tview.Pages, entry *libapt.SourceEntry, onDone func()) {
	form := styleForm(tview.NewForm())

	typeIdx := 0
	if entry.Type == "deb-src" {
		typeIdx = 1
	}
	form.AddDropDown("Type", []string{"deb", "deb-src"}, typeIdx, nil)
	form.AddInputField("URI", entry.URI, 60, nil, nil)
	form.AddInputField("Suite", entry.Suite, 40, nil, nil)
	form.AddInputField("Components", strings.Join(entry.Components, " "), 40, nil, nil)
	form.AddInputField("Options", entry.Options, 60, nil, nil)
	form.AddCheckbox("Enabled", entry.Enabled, nil)

	form.AddButton("Save", func() {
		_, typVal := form.GetFormItemByLabel("Type").(*tview.DropDown).GetCurrentOption()
		uri := strings.TrimSpace(form.GetFormItemByLabel("URI").(*tview.InputField).GetText())
		suite := strings.TrimSpace(form.GetFormItemByLabel("Suite").(*tview.InputField).GetText())
		comps := strings.TrimSpace(form.GetFormItemByLabel("Components").(*tview.InputField).GetText())
		opts := strings.TrimSpace(form.GetFormItemByLabel("Options").(*tview.InputField).GetText())
		enabled := form.GetFormItemByLabel("Enabled").(*tview.Checkbox).IsChecked()

		if uri == "" || suite == "" {
			showInfoModal(app, pages, "Missing fields", "URI and Suite are required.")
			return
		}

		updated := libapt.SourceEntry{
			Enabled:  enabled,
			Type:     typVal,
			URI:      uri,
			Suite:    suite,
			Options:  opts,
			FilePath: entry.FilePath,
		}
		if comps != "" {
			updated.Components = strings.Fields(comps)
		}

		err := libapt.EditSource(*entry, updated)
		if err != nil {
			showInfoModal(app, pages, "Error", fmt.Sprintf("Failed to edit source:\n%v", err))
			return
		}

		pages.RemovePage("edit-source")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("Cancel", func() {
		pages.RemovePage("edit-source")
	})

	form.SetBorder(true).SetTitle(" Edit Source ")
	pages.AddPage("edit-source", centered(form, 80, 20), true, true)
}

// showDeleteConfirm shows a confirmation modal for deleting a source.
func showDeleteConfirm(app *tview.Application, pages *tview.Pages, entry libapt.SourceEntry, onDone func()) {
	text := fmt.Sprintf("Delete this source?\n\n%s %s %s\nfrom %s",
		entry.Type, entry.URI, entry.Suite, filepath.Base(entry.FilePath))

	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.RemovePage("delete-confirm")
			if buttonLabel != "Delete" {
				return
			}

			err := libapt.DeleteSource(entry)
			if err != nil {
				showInfoModal(app, pages, "Error", fmt.Sprintf("Failed to delete source:\n%v", err))
				return
			}

			if onDone != nil {
				onDone()
			}
		})
	styleModal(modal)
	modal.SetTitle(" Confirm Delete ").SetBorder(true)
	pages.AddPage("delete-confirm", modal, true, true)
}

// showFileEditor shows a file picker and then a text editor for source files.
func showFileEditor(app *tview.Application, pages *tview.Pages, onDone func()) {
	files, err := libapt.ListSourceFiles()
	if err != nil {
		showInfoModal(app, pages, "Error", fmt.Sprintf("Failed to list source files:\n%v", err))
		return
	}

	if len(files) == 0 {
		showInfoModal(app, pages, "No files", "No APT source files found on this system.")
		return
	}

	fileList := styleList(tview.NewList().ShowSecondaryText(false))
	fileList.SetBorder(true).SetTitle(" Select Source File to Edit ")
	stylePanel(fileList.Box)

	for _, f := range files {
		filePath := f
		fileList.AddItem(filePath, "", 0, func() {
			openFileInEditor(app, pages, filePath, onDone)
		})
	}

	fileList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.RemovePage("file-picker")
			return nil
		}
		return event
	})

	pages.AddPage("file-picker", centered(fileList, 80, len(files)+4), true, true)
}

// openFileInEditor opens a source file in a TUI text editor.
func openFileInEditor(app *tview.Application, pages *tview.Pages, path string, onDone func()) {
	pages.RemovePage("file-picker")

	content, err := libapt.ReadSourceFile(path)
	if err != nil {
		showInfoModal(app, pages, "Error", fmt.Sprintf("Failed to read file:\n%v", err))
		return
	}

	editor := tview.NewTextArea()
	editor.SetText(content, true)
	editor.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", path))
	stylePanel(editor.Box)
	editor.SetTextStyle(tcell.StyleDefault.Foreground(colText))

	hints := []keyHint{
		{"ctrl+s", "save"},
		{"esc", "cancel"},
	}

	editorPage := chrome(editor, "Editing: "+filepath.Base(path), hints)

	editor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			newContent := editor.GetText()
			if newContent != content {
				modal := tview.NewModal().
					SetText("You have unsaved changes. Discard them?").
					AddButtons([]string{"Discard", "Keep Editing"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						pages.RemovePage("discard-confirm")
						if buttonLabel == "Discard" {
							pages.RemovePage("file-editor")
						}
					})
				styleModal(modal)
				modal.SetTitle(" Unsaved Changes ").SetBorder(true)
				pages.AddPage("discard-confirm", modal, true, true)
			} else {
				pages.RemovePage("file-editor")
			}
			return nil
		}

		if event.Key() == tcell.KeyCtrlS {
			if !ensureRoot(app, pages) {
				return nil
			}

			newContent := editor.GetText()
			err := libapt.WriteSourceFile(path, newContent)
			if err != nil {
				showInfoModal(app, pages, "Error", fmt.Sprintf("Failed to save file:\n%v", err))
				return nil
			}

			content = newContent

			modal := tview.NewModal().
				SetText("File saved successfully.").
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(_ int, _ string) {
					pages.RemovePage("save-confirm")
					pages.RemovePage("file-editor")
					if onDone != nil {
						onDone()
					}
				})
			styleModal(modal)
			modal.SetTitle(" Saved ").SetBorder(true)
			pages.AddPage("save-confirm", modal, true, true)
			return nil
		}

		return event
	})

	pages.AddPage("file-editor", editorPage, true, true)
	app.SetFocus(editor)
}
