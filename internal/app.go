package internal

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"gioui.org/f32"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type journalUI struct {
	cfg Config
	doc Document

	theme        *material.Theme
	editIcon     *widget.Icon
	previewIcon  *widget.Icon
	homeIcon     *widget.Icon
	settingsIcon *widget.Icon
	boldIcon     *widget.Icon
	italicIcon   *widget.Icon
	headingIcon  *widget.Icon
	listIcon     *widget.Icon
	linkIcon     *widget.Icon
	imageIcon    *widget.Icon
	tableIcon    *widget.Icon
	sectionIcon  *widget.Icon

	editor       widget.Editor
	previewList  widget.List
	settingsList widget.List
	homeList     widget.List
	find         widget.Editor
	replace      widget.Editor

	vaultPath   widget.Editor
	noteName    widget.Editor
	fontPath    widget.Editor
	bgColor     widget.Editor
	textColor   widget.Editor
	chromeColor widget.Editor
	accentColor widget.Editor

	newButton         widget.Clickable
	findButton        widget.Clickable
	findPrevButton    widget.Clickable
	findNextButton    widget.Clickable
	replaceButton     widget.Clickable
	replaceAllButton  widget.Clickable
	homeButton        widget.Clickable
	editButton        widget.Clickable
	previewButton     widget.Clickable
	settingsButton    widget.Clickable
	boldButton        widget.Clickable
	italicButton      widget.Clickable
	headingButton     widget.Clickable
	listButton        widget.Clickable
	linkButton        widget.Clickable
	imageButton       widget.Clickable
	tableButton       widget.Clickable
	sectionButton     widget.Clickable
	imagePickButton   widget.Clickable
	imageSmallButton  widget.Clickable
	imageMediumButton widget.Clickable
	imageLargeButton  widget.Clickable
	themePrevButton   widget.Clickable
	themeNextButton   widget.Clickable
	createButton      widget.Clickable
	vaultButton       widget.Clickable
	vaultApply        widget.Clickable
	fontButton        widget.Clickable
	fontApply         widget.Clickable

	motionToggle    widget.Bool
	toolbarToggle   widget.Bool
	homeStartToggle widget.Bool

	fontMinus    widget.Clickable
	fontPlus     widget.Clickable
	paddingMinus widget.Clickable
	paddingPlus  widget.Clickable
	widthMinus   widget.Clickable
	widthPlus    widget.Clickable

	showFind     bool
	showPreview  bool
	showSettings bool
	showHome     bool
	lastSaved    string
	status       string
	statusAt     time.Time
	toolbarY     float32
	findY        float32
	switchX      float32
	lastFrame    time.Time
	currentMatch int
	noteButtons  map[string]*widget.Clickable
	dayButtons   map[string]*widget.Clickable
	imageCache   map[string]previewImage
}

type previewImage struct {
	op   paint.ImageOp
	size image.Point
	err  error
}

func Run() error {
	errc := make(chan error, 1)
	go func() {
		errc <- runWindow()
	}()
	app.Main()
	return <-errc
}

func runWindow() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := ensureVault(cfg.Session.VaultPath); err != nil {
		return err
	}
	doc, err := loadDocument(cfg.Session.VaultPath, cfg.Session.LastNote)
	if err != nil {
		return err
	}

	var w app.Window
	w.Option(app.Title("JustJournal"))

	ui := newJournalUI(cfg, doc)
	var ops op.Ops
	for {
		switch ev := w.Event().(type) {
		case app.DestroyEvent:
			ui.flush()
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, ev)
			ui.layout(gtx)
			ev.Frame(gtx.Ops)
		}
	}
}

func newJournalUI(cfg Config, doc Document) *journalUI {
	ui := &journalUI{
		cfg:          cfg,
		doc:          doc,
		theme:        material.NewTheme(),
		editIcon:     iconFromData(icons.EditorModeEdit),
		previewIcon:  iconFromData(icons.ActionVisibility),
		homeIcon:     iconFromData(icons.ActionHome),
		settingsIcon: iconFromData(icons.ActionSettings),
		boldIcon:     iconFromData(icons.EditorFormatBold),
		italicIcon:   iconFromData(icons.EditorFormatItalic),
		headingIcon:  iconFromData(icons.EditorTitle),
		listIcon:     iconFromData(icons.EditorFormatListBulleted),
		linkIcon:     iconFromData(icons.EditorInsertLink),
		imageIcon:    iconFromData(icons.ImageImage),
		tableIcon:    iconFromData(icons.EditorBorderAll),
		sectionIcon:  iconFromData(icons.ActionViewHeadline),
		previewList:  widget.List{List: layout.List{Axis: layout.Vertical}},
		settingsList: widget.List{List: layout.List{Axis: layout.Vertical}},
		homeList:     widget.List{List: layout.List{Axis: layout.Vertical}},
		editor: widget.Editor{
			WrapPolicy: text.WrapGraphemes,
			InputHint:  key.HintText,
		},
		find:            widget.Editor{SingleLine: true},
		replace:         widget.Editor{SingleLine: true},
		vaultPath:       widget.Editor{SingleLine: true},
		noteName:        widget.Editor{SingleLine: true},
		fontPath:        widget.Editor{SingleLine: true},
		bgColor:         widget.Editor{SingleLine: true},
		textColor:       widget.Editor{SingleLine: true},
		chromeColor:     widget.Editor{SingleLine: true},
		accentColor:     widget.Editor{SingleLine: true},
		motionToggle:    widget.Bool{Value: cfg.Motion.Enabled},
		toolbarToggle:   widget.Bool{Value: cfg.UI.ToolbarVisible},
		homeStartToggle: widget.Bool{Value: cfg.UI.HomeOnStart},
		showHome:        cfg.UI.HomeOnStart,
		lastSaved:       doc.Content,
		noteButtons:     map[string]*widget.Clickable{},
		dayButtons:      map[string]*widget.Clickable{},
		imageCache:      map[string]previewImage{},
	}
	ui.reloadFonts()
	ui.editor.SetText(doc.Content)
	ui.syncSettingsEditors()
	return ui
}

func (ui *journalUI) layout(gtx layout.Context) layout.Dimensions {
	ui.applyThemePalette()
	if !ui.lastFrame.IsZero() {
		dt := gtx.Now.Sub(ui.lastFrame)
		target := float32(74)
		if !ui.showSettings && !ui.showHome && !ui.showPreview {
			target = 0
		}
		ui.toolbarY = expLerp(ui.toolbarY, target, ui.cfg.Motion.LerpRate, dt, ui.cfg.Motion.Enabled)
		findTarget := float32(-96)
		if ui.showFind {
			findTarget = 14
		}
		ui.findY = expLerp(ui.findY, findTarget, ui.cfg.Motion.LerpRate, dt, ui.cfg.Motion.Enabled)
		switchTarget := float32(0)
		if ui.showPreview {
			switchTarget = 34
		}
		ui.switchX = expLerp(ui.switchX, switchTarget, ui.cfg.Motion.LerpRate, dt, ui.cfg.Motion.Enabled)
		if ui.cfg.Motion.Enabled && (absFloat(ui.toolbarY-target) > 0.5 || absFloat(ui.findY-findTarget) > 0.5 || absFloat(ui.switchX-switchTarget) > 0.5) {
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(16 * time.Millisecond)})
		}
	} else if !ui.showFind {
		ui.findY = -96
		if ui.showPreview {
			ui.switchX = 34
		}
	}
	ui.lastFrame = gtx.Now
	ui.handleEvents(gtx)
	bg := parseColor(ui.cfg.Theme.Background, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(ui.toolbar),
		layout.Flexed(1, ui.bodyWithPopups),
		layout.Rigid(ui.statusBar),
	)
}

func (ui *journalUI) handleEvents(gtx layout.Context) {
	for ui.newButton.Clicked(gtx) {
		ui.createNote("")
	}
	for ui.findButton.Clicked(gtx) {
		ui.showFind = !ui.showFind
	}
	for ui.findPrevButton.Clicked(gtx) {
		ui.selectFindMatch(-1)
	}
	for ui.findNextButton.Clicked(gtx) {
		ui.selectFindMatch(1)
	}
	for ui.replaceButton.Clicked(gtx) {
		ui.replaceCurrentMatch()
	}
	for ui.replaceAllButton.Clicked(gtx) {
		ui.replaceAllMatches()
	}
	for ui.homeButton.Clicked(gtx) {
		ui.showHome = !ui.showHome
		ui.showSettings = false
	}
	for ui.editButton.Clicked(gtx) {
		ui.showPreview = false
		ui.showHome = false
		ui.switchX = 0
	}
	for ui.previewButton.Clicked(gtx) {
		ui.showPreview = true
		ui.showHome = false
		ui.switchX = 34
	}
	for ui.settingsButton.Clicked(gtx) {
		ui.showSettings = !ui.showSettings
		ui.showHome = false
	}
	for ui.boldButton.Clicked(gtx) {
		ui.wrapSelection("**", "**", "bold")
	}
	for ui.italicButton.Clicked(gtx) {
		ui.wrapSelection("*", "*", "italic")
	}
	for ui.headingButton.Clicked(gtx) {
		ui.insertHeading()
	}
	for ui.listButton.Clicked(gtx) {
		ui.editor.Insert("- ")
		ui.afterEditorCommand()
	}
	for ui.linkButton.Clicked(gtx) {
		ui.wrapSelection("[", "](https://)", "link")
	}
	for ui.imageButton.Clicked(gtx) {
		ui.editor.Insert("![image](path/to/image){w=480}")
		ui.afterEditorCommand()
	}
	for ui.tableButton.Clicked(gtx) {
		ui.insertTable()
	}
	for ui.sectionButton.Clicked(gtx) {
		ui.insertSection()
	}
	for ui.imagePickButton.Clicked(gtx) {
		ui.pickSelectedImagePath()
	}
	for ui.imageSmallButton.Clicked(gtx) {
		ui.applySelectedImageSize(240)
	}
	for ui.imageMediumButton.Clicked(gtx) {
		ui.applySelectedImageSize(480)
	}
	for ui.imageLargeButton.Clicked(gtx) {
		ui.applySelectedImageSize(720)
	}
	for ui.themePrevButton.Clicked(gtx) {
		ui.cycleThemePreset(-1)
	}
	for ui.themeNextButton.Clicked(gtx) {
		ui.cycleThemePreset(1)
	}
	for ui.createButton.Clicked(gtx) {
		ui.createNote(ui.noteName.Text())
	}
	for ui.vaultButton.Clicked(gtx) {
		if path, ok := nativeFolderDialog("Choose JustJournal vault", ui.cfg.Session.VaultPath); ok {
			ui.vaultPath.SetText(path)
			ui.applyVault(path)
		} else {
			ui.setStatus("Enter a vault path manually")
		}
	}
	for ui.vaultApply.Clicked(gtx) {
		ui.applyVault(ui.vaultPath.Text())
	}
	for ui.fontButton.Clicked(gtx) {
		if path, ok := nativeFileDialog("Choose font", ui.cfg.Typography.FontPath, []string{"Font files | *.ttf *.otf *.ttc"}); ok {
			ui.fontPath.SetText(path)
			ui.cfg.Typography.FontPath = path
			ui.reloadFonts()
			ui.saveConfigNow()
		} else {
			ui.setStatus("Enter a font path manually")
		}
	}
	for ui.fontApply.Clicked(gtx) {
		ui.cfg.Typography.FontPath = strings.TrimSpace(ui.fontPath.Text())
		ui.reloadFonts()
		ui.saveConfigNow()
	}
	if ui.motionToggle.Update(gtx) {
		ui.cfg.Motion.Enabled = ui.motionToggle.Value
		ui.saveConfigNow()
	}
	if ui.toolbarToggle.Update(gtx) {
		ui.cfg.UI.ToolbarVisible = ui.toolbarToggle.Value
		ui.saveConfigNow()
	}
	if ui.homeStartToggle.Update(gtx) {
		ui.cfg.UI.HomeOnStart = ui.homeStartToggle.Value
		ui.saveConfigNow()
	}
	for ui.fontMinus.Clicked(gtx) {
		ui.cfg.Typography.FontSize = clamp(ui.cfg.Typography.FontSize-1, 10, 36)
		ui.saveConfigNow()
	}
	for ui.fontPlus.Clicked(gtx) {
		ui.cfg.Typography.FontSize = clamp(ui.cfg.Typography.FontSize+1, 10, 36)
		ui.saveConfigNow()
	}
	for ui.paddingMinus.Clicked(gtx) {
		ui.cfg.Layout.PagePadding = clamp(ui.cfg.Layout.PagePadding-4, 8, 80)
		ui.saveConfigNow()
	}
	for ui.paddingPlus.Clicked(gtx) {
		ui.cfg.Layout.PagePadding = clamp(ui.cfg.Layout.PagePadding+4, 8, 80)
		ui.saveConfigNow()
	}
	for ui.widthMinus.Clicked(gtx) {
		ui.cfg.Layout.ContentWidth = clamp(ui.cfg.Layout.ContentWidth-40, 360, 1400)
		ui.saveConfigNow()
	}
	for ui.widthPlus.Clicked(gtx) {
		ui.cfg.Layout.ContentWidth = clamp(ui.cfg.Layout.ContentWidth+40, 360, 1400)
		ui.saveConfigNow()
	}

	for {
		ev, ok := ui.editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.ChangeEvent); ok {
			ui.doc.Content = ui.editor.Text()
			ui.saveDocumentNow()
		}
	}
	ui.updateSettingEditor(gtx, &ui.vaultPath, func(string) {})
	ui.updateSettingEditor(gtx, &ui.fontPath, func(string) {})
	ui.updateSettingEditor(gtx, &ui.bgColor, func(v string) {
		ui.cfg.Theme.Background = normalizeColor(v, ui.cfg.Theme.Background)
		ui.saveConfigNow()
	})
	ui.updateSettingEditor(gtx, &ui.textColor, func(v string) {
		ui.cfg.Theme.Text = normalizeColor(v, ui.cfg.Theme.Text)
		ui.saveConfigNow()
	})
	ui.updateSettingEditor(gtx, &ui.chromeColor, func(v string) {
		ui.cfg.Theme.Chrome = normalizeColor(v, ui.cfg.Theme.Chrome)
		ui.saveConfigNow()
	})
	ui.updateSettingEditor(gtx, &ui.accentColor, func(v string) {
		ui.cfg.Theme.Accent = normalizeColor(v, ui.cfg.Theme.Accent)
		ui.saveConfigNow()
	})
}

func (ui *journalUI) toolbar(gtx layout.Context) layout.Dimensions {
	chrome := parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xf7, G: 0xf7, B: 0xf5, A: 0xff})
	paint.FillShape(gtx.Ops, chrome, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(ui.cfg.UI.TopBarHeight)))}.Op())
	in := layout.Inset{Left: 10, Right: 10, Top: 5, Bottom: 5}
	return in.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.newButton, "New") }),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.findButton, "Find") }),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.iconButton(gtx, &ui.homeButton, ui.homeIcon, "Home")
			}),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.iconButton(gtx, &ui.settingsButton, ui.settingsIcon, "Settings")
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(ui.viewSwitch),
			layout.Rigid(spacer(12, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !ui.cfg.UI.ShowNoteName {
					return layout.Dimensions{}
				}
				return ui.label(gtx, filepath.Base(ui.doc.Path), unit.Sp(13), mutedColor(ui.cfg), text.End)
			}),
		)
	})
}

func (ui *journalUI) bodyWithPopups(gtx layout.Context) layout.Dimensions {
	return layout.Stack{Alignment: layout.N}.Layout(gtx,
		layout.Expanded(ui.body),
		layout.Stacked(ui.findPopup),
	)
}

func (ui *journalUI) findPopup(gtx layout.Context) layout.Dimensions {
	if !ui.showFind && ui.findY <= -90 {
		return layout.Dimensions{}
	}
	defer op.Offset(image.Pt(0, int(ui.findY))).Push(gtx.Ops).Pop()
	return layout.Inset{Top: 0}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = minInt(gtx.Constraints.Max.X, gtx.Dp(620))
		return widget.Border{
			Color:        color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x22},
			CornerRadius: 16,
			Width:        1,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					paint.FillShape(gtx.Ops, color.NRGBA{R: 0x1c, G: 0x1c, B: 0x1e, A: 0xee}, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(16)).Op(gtx.Ops))
					return layout.Dimensions{Size: gtx.Constraints.Min}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: 12, Right: 12, Top: 10, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return ui.editorField(gtx, &ui.find, "Find in note")
									}),
									layout.Rigid(spacer(8, 1)),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.darkButton(gtx, &ui.findPrevButton, "Prev") }),
									layout.Rigid(spacer(6, 1)),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.darkButton(gtx, &ui.findNextButton, "Next") }),
									layout.Rigid(spacer(10, 1)),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, findPositionLabel(ui.editor.Text(), ui.find.Text(), ui.currentMatch), 12, color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf7, A: 0xff}, text.End)
									}),
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return ui.editorField(gtx, &ui.replace, "Replace with")
										}),
										layout.Rigid(spacer(8, 1)),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.darkButton(gtx, &ui.replaceButton, "Replace") }),
										layout.Rigid(spacer(6, 1)),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.darkButton(gtx, &ui.replaceAllButton, "All") }),
									)
								})
							}),
						)
					})
				},
			)
		})
	})
}

func (ui *journalUI) body(gtx layout.Context) layout.Dimensions {
	if ui.showSettings {
		return ui.settings(gtx)
	}
	if ui.showHome {
		return ui.home(gtx)
	}
	pad := unit.Dp(ui.cfg.Layout.PagePadding)
	return layout.UniformInset(pad).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = minInt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(ui.cfg.Layout.ContentWidth)))
		if ui.showPreview {
			return ui.preview(gtx)
		}
		return ui.editorWithToolbar(gtx)
	})
}

func (ui *journalUI) editorWithToolbar(gtx layout.Context) layout.Dimensions {
	if !ui.cfg.UI.ToolbarVisible {
		return ui.editorPane(gtx)
	}
	return layout.Stack{Alignment: layout.S}.Layout(gtx,
		layout.Expanded(ui.editorPane),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 84}.Layout(gtx, ui.selectedImagePathPopup)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			defer op.Offset(image.Pt(0, int(ui.toolbarY))).Push(gtx.Ops).Pop()
			return layout.Inset{Bottom: 22}.Layout(gtx, ui.formatToolbar)
		}),
	)
}

func (ui *journalUI) selectedImagePathPopup(gtx layout.Context) layout.Dimensions {
	selected := strings.TrimSpace(ui.editor.SelectedText())
	if selected == "" || !looksLikeImagePath(selected) {
		return layout.Dimensions{}
	}
	return widget.Border{
		Color:        color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x22},
		CornerRadius: 14,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, color.NRGBA{R: 0x1c, G: 0x1c, B: 0x1e, A: 0xee}, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(14)).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: 10, Right: 10, Top: 8, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "Image path", 12, color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf7, A: 0xff}, text.Start)
						}),
						layout.Rigid(spacer(10, 1)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.imagePickButton, "Pick file")
						}),
						layout.Rigid(spacer(8, 1)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.darkButton(gtx, &ui.imageSmallButton, "Small")
						}),
						layout.Rigid(spacer(6, 1)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.darkButton(gtx, &ui.imageMediumButton, "Med")
						}),
						layout.Rigid(spacer(6, 1)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.darkButton(gtx, &ui.imageLargeButton, "Large")
						}),
					)
				})
			},
		)
	})
}

func (ui *journalUI) editorPane(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	style := material.Editor(ui.theme, &ui.editor, "Write markdown")
	style.TextSize = unit.Sp(ui.cfg.Typography.FontSize)
	style.Color = parseColor(ui.cfg.Theme.Text, color.NRGBA{R: 0x20, G: 0x21, B: 0x24, A: 0xff})
	style.HintColor = mutedColor(ui.cfg)
	style.SelectionColor = parseColor(ui.cfg.Theme.Accent, color.NRGBA{R: 0xdc, G: 0xe6, B: 0xff, A: 0xff})
	return layout.NW.Layout(gtx, style.Layout)
}

func (ui *journalUI) preview(gtx layout.Context) layout.Dimensions {
	blocks := markdownBlocks(ui.editor.Text(), ui.cfg, filepath.Dir(ui.doc.Path))
	return ui.softList(&ui.previewList).Layout(gtx, len(blocks), func(gtx layout.Context, i int) layout.Dimensions {
		b := blocks[i]
		return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.previewBlock(gtx, b)
		})
	})
}

func (ui *journalUI) home(gtx layout.Context) layout.Dimensions {
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "JustJournal", 28, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Start)
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 6, Bottom: 18}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, "Pick a note or start from a day.", 14, mutedColor(ui.cfg), text.Start)
			})
		},
		ui.calendar,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 18, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, "Recent", 16, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Start)
			})
		},
		ui.recentNotes,
	}
	return layout.UniformInset(unit.Dp(ui.cfg.Layout.PagePadding)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = minInt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(ui.cfg.Layout.ContentWidth)))
		return ui.softList(&ui.homeList).LayoutWidgets(gtx, rows...)
	})
}

func (ui *journalUI) settings(gtx layout.Context) layout.Dimensions {
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return ui.sectionLabel(gtx, "Note") },
		func(gtx layout.Context) layout.Dimensions {
			return ui.settingRow(gtx, "Name", func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.editorField(gtx, &ui.noteName, "note.md") }),
					layout.Rigid(spacer(8, 1)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.createButton, "Create") }),
				)
			})
		},
		func(gtx layout.Context) layout.Dimensions { return ui.sectionLabel(gtx, "Theme") },
		ui.themePresetRow,
		func(gtx layout.Context) layout.Dimensions { return ui.colorRow(gtx, "Background", &ui.bgColor) },
		func(gtx layout.Context) layout.Dimensions { return ui.colorRow(gtx, "Text", &ui.textColor) },
		func(gtx layout.Context) layout.Dimensions { return ui.colorRow(gtx, "Chrome", &ui.chromeColor) },
		func(gtx layout.Context) layout.Dimensions { return ui.colorRow(gtx, "Accent", &ui.accentColor) },
		func(gtx layout.Context) layout.Dimensions { return ui.sectionLabel(gtx, "Layout") },
		func(gtx layout.Context) layout.Dimensions {
			return ui.stepperRow(gtx, "Font size", fmt.Sprintf("%.0f", ui.cfg.Typography.FontSize), &ui.fontMinus, &ui.fontPlus)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.stepperRow(gtx, "Padding", fmt.Sprintf("%.0f", ui.cfg.Layout.PagePadding), &ui.paddingMinus, &ui.paddingPlus)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.stepperRow(gtx, "Width", fmt.Sprintf("%.0f", ui.cfg.Layout.ContentWidth), &ui.widthMinus, &ui.widthPlus)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.settingRow(gtx, "Motion", func(gtx layout.Context) layout.Dimensions {
				return material.CheckBox(ui.theme, &ui.motionToggle, "Enabled").Layout(gtx)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.settingRow(gtx, "Toolbar", func(gtx layout.Context) layout.Dimensions {
				return material.CheckBox(ui.theme, &ui.toolbarToggle, "Visible").Layout(gtx)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.settingRow(gtx, "Start", func(gtx layout.Context) layout.Dimensions {
				return material.CheckBox(ui.theme, &ui.homeStartToggle, "Home screen").Layout(gtx)
			})
		},
		func(gtx layout.Context) layout.Dimensions { return ui.sectionLabel(gtx, "Files") },
		func(gtx layout.Context) layout.Dimensions {
			return ui.pathRow(gtx, "Vault", &ui.vaultPath, &ui.vaultButton, &ui.vaultApply)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.pathRow(gtx, "Font", &ui.fontPath, &ui.fontButton, &ui.fontApply)
		},
	}
	return layout.UniformInset(unit.Dp(ui.cfg.Layout.PagePadding)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = minInt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(ui.cfg.Layout.ContentWidth)))
		return ui.settingsSurface(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.softList(&ui.settingsList).LayoutWidgets(gtx, rows...)
		})
	})
}

func (ui *journalUI) softList(list *widget.List) material.ListStyle {
	style := material.List(ui.theme, list)
	style.AnchorStrategy = material.Overlay
	style.Track.Color = color.NRGBA{}
	style.Track.MajorPadding = 10
	style.Track.MinorPadding = 4
	style.Indicator.MinorWidth = 3
	style.Indicator.CornerRadius = 2
	style.Indicator.MajorMinLen = 34
	thumb := parseColor(ui.cfg.Theme.Text, color.NRGBA{R: 0x20, G: 0x21, B: 0x24, A: 0xff})
	thumb.A = 72
	hover := thumb
	hover.A = 138
	style.Indicator.Color = thumb
	style.Indicator.HoverColor = hover
	return style
}

func (ui *journalUI) settingsSurface(gtx layout.Context, content layout.Widget) layout.Dimensions {
	return widget.Border{
		Color:        parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}),
		CornerRadius: 16,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				bg := parseColor(ui.cfg.Theme.Background, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
				paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(16)).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(18).Layout(gtx, content)
			},
		)
	})
}

func (ui *journalUI) settingRow(gtx layout.Context, name string, value layout.Widget) layout.Dimensions {
	return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(120)
				return ui.label(gtx, name, 14, mutedColor(ui.cfg), text.Start)
			}),
			layout.Flexed(1, value),
		)
	})
}

func (ui *journalUI) colorRow(gtx layout.Context, name string, editor *widget.Editor) layout.Dimensions {
	return ui.settingRow(gtx, name, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.colorSwatch(gtx, editor.Text())
			}),
			layout.Rigid(spacer(10, 1)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return ui.editorField(gtx, editor, "#RRGGBB")
			}),
			layout.Rigid(spacer(10, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, rgbLabel(editor.Text()), 12, mutedColor(ui.cfg), text.Start)
			}),
		)
	})
}

func (ui *journalUI) themePresetRow(gtx layout.Context) layout.Dimensions {
	preset := ui.currentThemePreset()
	return ui.settingRow(gtx, "Preset", func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &ui.themePrevButton, "Prev")
			}),
			layout.Rigid(spacer(8, 1)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return widget.Border{
					Color:        parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}),
					CornerRadius: 10,
					Width:        1,
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: 12, Right: 12, Top: 8, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.paletteDots(gtx, preset.Theme)
							}),
							layout.Rigid(spacer(10, 1)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, preset.Name, 13, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Start)
							}),
						)
					})
				})
			}),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &ui.themeNextButton, "Next")
			}),
		)
	})
}

func (ui *journalUI) currentThemePreset() themePreset {
	presets := themePresets()
	for _, preset := range presets {
		if sameTheme(ui.cfg.Theme, preset.Theme) {
			return preset
		}
	}
	return themePreset{Name: "Custom", Theme: ui.cfg.Theme}
}

func (ui *journalUI) cycleThemePreset(delta int) {
	presets := themePresets()
	index := 0
	for i, preset := range presets {
		if sameTheme(ui.cfg.Theme, preset.Theme) {
			index = i
			break
		}
	}
	index = (index + delta + len(presets)) % len(presets)
	ui.applyThemePreset(presets[index])
}

func sameTheme(a, b ThemeConfig) bool {
	return strings.EqualFold(a.Background, b.Background) &&
		strings.EqualFold(a.Text, b.Text) &&
		strings.EqualFold(a.Chrome, b.Chrome) &&
		strings.EqualFold(a.Accent, b.Accent) &&
		strings.EqualFold(a.Muted, b.Muted)
}

func (ui *journalUI) paletteDots(gtx layout.Context, theme ThemeConfig) layout.Dimensions {
	colors := []string{theme.Background, theme.Text, theme.Accent}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return colorDot(gtx, colors[0]) }),
		layout.Rigid(spacer(3, 1)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return colorDot(gtx, colors[1]) }),
		layout.Rigid(spacer(3, 1)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return colorDot(gtx, colors[2]) }),
	)
}

func colorDot(gtx layout.Context, value string) layout.Dimensions {
	size := image.Pt(gtx.Dp(9), gtx.Dp(9))
	paint.FillShape(gtx.Ops, parseColor(value, color.NRGBA{A: 0xff}), clip.Ellipse(image.Rectangle{Max: size}).Op(gtx.Ops))
	return layout.Dimensions{Size: size}
}

func (ui *journalUI) colorSwatch(gtx layout.Context, value string) layout.Dimensions {
	size := image.Pt(gtx.Dp(28), gtx.Dp(28))
	col := parseColor(value, mutedColor(ui.cfg))
	paint.FillShape(gtx.Ops, col, clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(8)).Op(gtx.Ops))
	return layout.Dimensions{Size: size}
}

func (ui *journalUI) pathRow(gtx layout.Context, name string, editor *widget.Editor, choose, apply *widget.Clickable) layout.Dimensions {
	return ui.settingRow(gtx, name, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.editorField(gtx, editor, "") }),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, choose, "Choose") }),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, apply, "Apply") }),
		)
	})
}

func (ui *journalUI) stepperRow(gtx layout.Context, name, value string, minus, plus *widget.Clickable) layout.Dimensions {
	return ui.settingRow(gtx, name, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, minus, "-") }),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(60)
				return ui.label(gtx, value, 14, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Middle)
			}),
			layout.Rigid(spacer(8, 1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, plus, "+") }),
		)
	})
}

func (ui *journalUI) sectionLabel(gtx layout.Context, content string) layout.Dimensions {
	return layout.Inset{Top: 12, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return ui.label(gtx, content, 16, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Start)
	})
}

func (ui *journalUI) statusBar(gtx layout.Context) layout.Dimensions {
	msg := ui.status
	if msg == "" || time.Since(ui.statusAt) > 4*time.Second {
		msg = ui.doc.Path
	}
	return layout.Inset{Left: 10, Right: 10, Top: 4, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return ui.label(gtx, msg, 12, mutedColor(ui.cfg), text.Start)
	})
}

func (ui *journalUI) button(gtx layout.Context, click *widget.Clickable, label string) layout.Dimensions {
	style := material.Button(ui.theme, click, label)
	style.Background = parseColor(ui.cfg.Theme.Accent, color.NRGBA{R: 0xdc, G: 0xe6, B: 0xff, A: 0xff})
	style.Color = parseColor(ui.cfg.Theme.Text, color.NRGBA{R: 0x20, G: 0x21, B: 0x24, A: 0xff})
	return style.Layout(gtx)
}

func (ui *journalUI) darkButton(gtx layout.Context, click *widget.Clickable, label string) layout.Dimensions {
	style := material.Button(ui.theme, click, label)
	style.Background = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1f}
	style.Color = color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf7, A: 0xff}
	style.TextSize = 12
	style.Inset = layout.Inset{Left: 9, Right: 9, Top: 7, Bottom: 7}
	return style.Layout(gtx)
}

func (ui *journalUI) iconButton(gtx layout.Context, click *widget.Clickable, icon *widget.Icon, title string) layout.Dimensions {
	style := material.IconButton(ui.theme, click, icon, title)
	style.Background = parseColor(ui.cfg.Theme.Accent, color.NRGBA{R: 0xdc, G: 0xe6, B: 0xff, A: 0xff})
	style.Color = parseColor(ui.cfg.Theme.Text, color.NRGBA{R: 0x20, G: 0x21, B: 0x24, A: 0xff})
	style.Size = 18
	style.Inset = layout.UniformInset(8)
	return style.Layout(gtx)
}

func (ui *journalUI) formatToolbar(gtx layout.Context) layout.Dimensions {
	return widget.Border{
		Color:        color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x22},
		CornerRadius: 18,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, color.NRGBA{R: 0x1c, G: 0x1c, B: 0x1e, A: 0xe8}, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(18)).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.boldButton, ui.boldIcon, "Bold")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.italicButton, ui.italicIcon, "Italic")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.headingButton, ui.headingIcon, "Heading")
						}),
						layout.Rigid(ui.toolDivider),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.listButton, ui.listIcon, "List")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.linkButton, ui.linkIcon, "Link")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.tableButton, ui.tableIcon, "Table")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.sectionButton, ui.sectionIcon, "Section")
						}),
						layout.Rigid(ui.toolDivider),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.toolButton(gtx, &ui.imageButton, ui.imageIcon, "Image")
						}),
					)
				})
			},
		)
	})
}

func (ui *journalUI) toolButton(gtx layout.Context, click *widget.Clickable, icon *widget.Icon, title string) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Pt(gtx.Dp(36), gtx.Dp(36))
		bg := color.NRGBA{}
		if click.Hovered() {
			bg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1f}
		}
		paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(10)).Op(gtx.Ops))
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = image.Pt(gtx.Dp(18), gtx.Dp(18))
			if icon == nil {
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}
			return icon.Layout(gtx, color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf7, A: 0xff})
		})
	})
}

func (ui *journalUI) toolDivider(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: 4, Right: 4, Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(gtx.Dp(1), gtx.Dp(24))
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x26}, clip.Rect{Max: size}.Op())
		return layout.Dimensions{Size: size}
	})
}

func (ui *journalUI) viewSwitch(gtx layout.Context) layout.Dimensions {
	return widget.Border{
		Color:        parseColor(ui.cfg.Theme.Accent, color.NRGBA{R: 0xdc, G: 0xe6, B: 0xff, A: 0xff}),
		CornerRadius: 6,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Pt(gtx.Dp(68), gtx.Dp(30))
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				bg := parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xf7, G: 0xf7, B: 0xf5, A: 0xff})
				paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(5)).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				size := image.Pt(gtx.Dp(34), gtx.Dp(30))
				defer op.Offset(image.Pt(int(ui.switchX), 0)).Push(gtx.Ops).Pop()
				paint.FillShape(gtx.Ops, parseColor(ui.cfg.Theme.Accent, color.NRGBA{R: 0xdc, G: 0xe6, B: 0xff, A: 0xff}), clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(5)).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.switchButton(gtx, &ui.editButton, ui.editIcon)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.switchButton(gtx, &ui.previewButton, ui.previewIcon)
					}),
				)
			}),
		)
	})
}

func (ui *journalUI) switchButton(gtx layout.Context, click *widget.Clickable, icon *widget.Icon) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Dp(34)
		gtx.Constraints.Min.Y = gtx.Dp(30)
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = image.Pt(gtx.Dp(18), gtx.Dp(18))
			if icon == nil {
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}
			return icon.Layout(gtx, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}))
		})
	})
}

func (ui *journalUI) editorField(gtx layout.Context, editor *widget.Editor, hint string) layout.Dimensions {
	style := material.Editor(ui.theme, editor, hint)
	style.TextSize = 14
	style.Color = parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff})
	style.HintColor = mutedColor(ui.cfg)
	return widget.Border{
		Color:        parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xdd, G: 0xdd, B: 0xdd, A: 0xff}),
		CornerRadius: 4,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(6).Layout(gtx, style.Layout)
	})
}

func (ui *journalUI) label(gtx layout.Context, content string, size unit.Sp, col color.NRGBA, align text.Alignment) layout.Dimensions {
	style := material.Label(ui.theme, size, content)
	style.Color = col
	style.Alignment = align
	return style.Layout(gtx)
}

func (ui *journalUI) styledLabel(gtx layout.Context, content string, size unit.Sp, col color.NRGBA, align text.Alignment, f font.Font) layout.Dimensions {
	style := material.Label(ui.theme, size, content)
	style.Color = col
	style.Alignment = align
	style.Font = f
	style.WrapPolicy = text.WrapWords
	draw := func(gtx layout.Context) layout.Dimensions {
		if f.Style == font.Italic {
			defer op.Affine(f32.AffineId().Shear(f32.Point{}, -0.18, 0)).Push(gtx.Ops).Pop()
		}
		return style.Layout(gtx)
	}
	if f.Weight >= font.Bold {
		dims := draw(gtx)
		defer op.Offset(image.Pt(1, 0)).Push(gtx.Ops).Pop()
		draw(gtx)
		return dims
	}
	return draw(gtx)
}

func (ui *journalUI) previewBlock(gtx layout.Context, block previewBlock) layout.Dimensions {
	if len(block.Table) > 0 {
		return ui.previewTable(gtx, block)
	}
	if block.ImagePath != "" {
		return ui.previewImage(gtx, block)
	}
	if len(block.Segments) == 0 {
		return ui.styledLabel(gtx, block.Text, block.Size, block.Color, text.Start, block.Font)
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx, block.segmentWidgets(ui)...)
}

func (ui *journalUI) previewTable(gtx layout.Context, block previewBlock) layout.Dimensions {
	cols := tableColumnCount(block.Table)
	if cols == 0 {
		return layout.Dimensions{}
	}
	return widget.Border{
		Color:        parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}),
		CornerRadius: 10,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, tableRowWidgets(ui, block.Table, cols)...)
	})
}

func (ui *journalUI) previewImage(gtx layout.Context, block previewBlock) layout.Dimensions {
	img := ui.loadPreviewImage(block.ImagePath)
	if img.err != nil || img.size.X <= 0 || img.size.Y <= 0 {
		msg := block.Text
		if img.err != nil {
			msg += " (" + img.err.Error() + ")"
		}
		return ui.label(gtx, msg, 14, mutedColor(ui.cfg), text.Start)
	}
	maxW := gtx.Constraints.Max.X
	if block.ImageWidth > 0 {
		maxW = minInt(maxW, gtx.Dp(unit.Dp(block.ImageWidth)))
	}
	scale := float32(1)
	if img.size.X > maxW {
		scale = float32(maxW) / float32(img.size.X)
	}
	out := image.Pt(int(float32(img.size.X)*scale), int(float32(img.size.Y)*scale))
	return widget.Border{
		Color:        parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}),
		CornerRadius: 12,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		defer clip.UniformRRect(image.Rectangle{Max: out}, gtx.Dp(12)).Push(gtx.Ops).Pop()
		defer op.Affine(f32.AffineId().Scale(f32.Point{}, f32.Pt(scale, scale))).Push(gtx.Ops).Pop()
		img.op.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		return layout.Dimensions{Size: out}
	})
}

func (ui *journalUI) loadPreviewImage(path string) previewImage {
	if img, ok := ui.imageCache[path]; ok {
		return img
	}
	file, err := os.Open(path)
	if err != nil {
		img := previewImage{err: err}
		ui.imageCache[path] = img
		return img
	}
	defer file.Close()
	decoded, _, err := image.Decode(file)
	if err != nil {
		img := previewImage{err: err}
		ui.imageCache[path] = img
		return img
	}
	op := paint.NewImageOp(decoded)
	img := previewImage{op: op, size: op.Size()}
	ui.imageCache[path] = img
	return img
}

func (ui *journalUI) createNote(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = time.Now().Format("2006-01-02") + ".md"
	}
	ui.flush()
	doc, err := loadDocument(ui.cfg.Session.VaultPath, name)
	if err != nil {
		ui.setStatus(userFacingError(err))
		return
	}
	ui.doc = doc
	ui.cfg.Session.LastNote = filepath.Base(doc.Path)
	ui.editor.SetText(doc.Content)
	ui.noteName.SetText(filepath.Base(doc.Path))
	ui.lastSaved = doc.Content
	ui.saveConfigNow()
	ui.setStatus("Opened " + filepath.Base(doc.Path))
}

func (ui *journalUI) openNote(name string) {
	ui.flush()
	doc, err := loadDocument(ui.cfg.Session.VaultPath, name)
	if err != nil {
		ui.setStatus(userFacingError(err))
		return
	}
	ui.doc = doc
	ui.cfg.Session.LastNote = filepath.Base(doc.Path)
	ui.editor.SetText(doc.Content)
	ui.noteName.SetText(filepath.Base(doc.Path))
	ui.lastSaved = doc.Content
	ui.showHome = false
	ui.showPreview = false
	ui.saveConfigNow()
	ui.setStatus("Opened " + filepath.Base(doc.Path))
}

func (ui *journalUI) applyVault(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	ui.flush()
	ui.cfg.Session.VaultPath = path
	doc, err := loadDocument(path, filepath.Base(ui.doc.Path))
	if err != nil {
		ui.setStatus(userFacingError(err))
		return
	}
	ui.doc = doc
	ui.cfg.Session.LastNote = filepath.Base(doc.Path)
	ui.editor.SetText(doc.Content)
	ui.lastSaved = doc.Content
	ui.saveConfigNow()
	ui.setStatus("Vault set to " + path)
}

func (ui *journalUI) flush() {
	ui.doc.Content = ui.editor.Text()
	_ = saveDocument(ui.doc)
	_ = saveConfig(ui.cfg)
}

func (ui *journalUI) saveDocumentNow() {
	if ui.doc.Content == ui.lastSaved {
		return
	}
	if err := saveDocument(ui.doc); err != nil {
		ui.setStatus(userFacingError(err))
		return
	}
	ui.lastSaved = ui.doc.Content
	ui.setStatus("Saved")
}

func (ui *journalUI) saveConfigNow() {
	if err := saveConfig(ui.cfg); err != nil {
		ui.setStatus(userFacingError(err))
		return
	}
	ui.setStatus("Settings saved")
}

func (ui *journalUI) applyThemePreset(preset themePreset) {
	ui.cfg.Theme = preset.Theme
	ui.syncSettingsEditors()
	ui.saveConfigNow()
}

type themePreset struct {
	Name  string
	Theme ThemeConfig
}

func themePresets() []themePreset {
	defaultTheme := defaultConfig().Theme
	return []themePreset{
		{Name: "Default", Theme: defaultTheme},
		{Name: "Apple Light", Theme: ThemeConfig{Background: "#F5F5F7", Text: "#1D1D1F", Chrome: "#FFFFFF", Accent: "#007AFF", Muted: "#6E6E73"}},
		{Name: "Apple Dark", Theme: ThemeConfig{Background: "#1C1C1E", Text: "#F5F5F7", Chrome: "#2C2C2E", Accent: "#0A84FF", Muted: "#A1A1AA"}},
		{Name: "Catppuccin", Theme: ThemeConfig{Background: "#1E1E2E", Text: "#CDD6F4", Chrome: "#313244", Accent: "#89B4FA", Muted: "#A6ADC8"}},
		{Name: "GitHub Light", Theme: ThemeConfig{Background: "#FFFFFF", Text: "#24292F", Chrome: "#F6F8FA", Accent: "#0969DA", Muted: "#57606A"}},
		{Name: "GitHub Dark", Theme: ThemeConfig{Background: "#0D1117", Text: "#E6EDF3", Chrome: "#161B22", Accent: "#2F81F7", Muted: "#8B949E"}},
	}
}

func (ui *journalUI) reloadFonts() {
	collection := gofont.Collection()
	path := strings.TrimSpace(ui.cfg.Typography.FontPath)
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			ui.setStatus(userFacingError(err))
		} else if faces, err := opentype.ParseCollection(data); err != nil {
			ui.setStatus(userFacingError(err))
		} else {
			collection = append(faces, collection...)
		}
	}
	ui.theme.Shaper = text.NewShaper(text.WithCollection(collection))
}

func (ui *journalUI) applyThemePalette() {
	ui.theme.Palette.Bg = parseColor(ui.cfg.Theme.Background, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	ui.theme.Palette.Fg = parseColor(ui.cfg.Theme.Text, color.NRGBA{R: 0x20, G: 0x21, B: 0x24, A: 0xff})
	ui.theme.Palette.ContrastBg = parseColor(ui.cfg.Theme.Accent, color.NRGBA{R: 0xdc, G: 0xe6, B: 0xff, A: 0xff})
	ui.theme.Palette.ContrastFg = parseColor(ui.cfg.Theme.Text, color.NRGBA{R: 0x20, G: 0x21, B: 0x24, A: 0xff})
}

func (ui *journalUI) updateSettingEditor(gtx layout.Context, editor *widget.Editor, apply func(string)) {
	for {
		ev, ok := editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.ChangeEvent); ok {
			apply(editor.Text())
		}
	}
}

func (ui *journalUI) syncSettingsEditors() {
	ui.vaultPath.SetText(ui.cfg.Session.VaultPath)
	ui.noteName.SetText(filepath.Base(ui.doc.Path))
	ui.fontPath.SetText(ui.cfg.Typography.FontPath)
	ui.bgColor.SetText(ui.cfg.Theme.Background)
	ui.textColor.SetText(ui.cfg.Theme.Text)
	ui.chromeColor.SetText(ui.cfg.Theme.Chrome)
	ui.accentColor.SetText(ui.cfg.Theme.Accent)
	ui.motionToggle.Value = ui.cfg.Motion.Enabled
	ui.toolbarToggle.Value = ui.cfg.UI.ToolbarVisible
	ui.homeStartToggle.Value = ui.cfg.UI.HomeOnStart
}

func (ui *journalUI) setStatus(msg string) {
	ui.status = msg
	ui.statusAt = time.Now()
}

func (ui *journalUI) wrapSelection(prefix, suffix, placeholder string) {
	selected := ui.editor.SelectedText()
	if selected == "" {
		selected = placeholder
	}
	if ui.editor.SelectionLen() > 0 {
		ui.editor.Delete(1)
	}
	ui.editor.Insert(prefix + selected + suffix)
	ui.afterEditorCommand()
}

func (ui *journalUI) insertHeading() {
	line, _ := ui.editor.CaretPos()
	if line == 0 && strings.TrimSpace(ui.editor.Text()) == "" {
		ui.editor.Insert("# ")
	} else {
		ui.editor.Insert("\n# ")
	}
	ui.afterEditorCommand()
}

func (ui *journalUI) insertSection() {
	ui.editor.Insert("\n---\n\n## Section\n\n")
	ui.afterEditorCommand()
}

func (ui *journalUI) insertTable() {
	ui.editor.Insert("\n| Column | Value |\n| --- | --- |\n| Item | Detail |\n\n")
	ui.afterEditorCommand()
}

func (ui *journalUI) afterEditorCommand() {
	ui.doc.Content = ui.editor.Text()
	ui.saveDocumentNow()
}

func (ui *journalUI) selectFindMatch(direction int) {
	matches := findMatchRanges(ui.editor.Text(), ui.find.Text())
	if len(matches) == 0 {
		ui.currentMatch = 0
		ui.setStatus("No matches")
		return
	}
	start, _ := ui.editor.Selection()
	index := matchIndexAtOrAfter(matches, start)
	if direction < 0 {
		index--
	} else if direction > 0 {
		if index < len(matches) && matches[index].start == start {
			index++
		}
	}
	if index < 0 {
		index = len(matches) - 1
	}
	if index >= len(matches) {
		index = 0
	}
	ui.currentMatch = index
	ui.editor.SetCaret(matches[index].start, matches[index].end)
	ui.showPreview = false
	ui.showHome = false
}

func (ui *journalUI) replaceCurrentMatch() {
	matches := findMatchRanges(ui.editor.Text(), ui.find.Text())
	if len(matches) == 0 {
		ui.setStatus("No matches")
		return
	}
	start, end := ui.editor.Selection()
	index := exactMatchIndex(matches, start, end)
	if index < 0 {
		ui.selectFindMatch(1)
		start, end = ui.editor.Selection()
		index = exactMatchIndex(matches, start, end)
		if index < 0 {
			return
		}
	}
	ui.editor.Delete(1)
	ui.editor.Insert(ui.replace.Text())
	ui.afterEditorCommand()
	ui.selectFindMatch(1)
}

func (ui *journalUI) replaceAllMatches() {
	find := ui.find.Text()
	if find == "" {
		return
	}
	text := ui.editor.Text()
	next, count := replaceAllCaseInsensitive(text, find, ui.replace.Text())
	if count == 0 {
		ui.setStatus("No matches")
		return
	}
	ui.editor.SetText(next)
	ui.editor.SetCaret(0, 0)
	ui.currentMatch = 0
	ui.afterEditorCommand()
	ui.setStatus(fmt.Sprintf("Replaced %d", count))
}

func (ui *journalUI) applySelectedImageSize(width int) {
	selected := strings.TrimSpace(ui.editor.SelectedText())
	if selected == "" {
		return
	}
	path := extractImagePath(selected)
	if path == "" {
		path = selected
	}
	replacement := fmt.Sprintf("![image](%s){w=%d}", strings.Trim(path, `"'`), width)
	if ui.editor.SelectionLen() > 0 {
		ui.editor.Delete(1)
	}
	ui.editor.Insert(replacement)
	ui.afterEditorCommand()
}

func (ui *journalUI) pickSelectedImagePath() {
	selected := strings.TrimSpace(ui.editor.SelectedText())
	if selected == "" {
		return
	}
	current := selected
	if !filepath.IsAbs(current) {
		current = filepath.Join(filepath.Dir(ui.doc.Path), current)
	}
	path, ok := nativeFileDialog("Choose image", current, []string{"Image files | *.png *.jpg *.jpeg *.gif"})
	if !ok {
		ui.setStatus("Enter an image path manually")
		return
	}
	replacement := path
	if rel, err := filepath.Rel(filepath.Dir(ui.doc.Path), path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		replacement = rel
	}
	if ui.editor.SelectionLen() > 0 {
		ui.editor.Delete(1)
	}
	ui.editor.Insert(replacement)
	ui.imageCache = map[string]previewImage{}
	ui.afterEditorCommand()
}

func (ui *journalUI) calendar(gtx layout.Context) layout.Dimensions {
	now := time.Now()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	offset := int(first.Weekday())
	days := daysInMonth(now.Year(), now.Month())
	title := now.Format("June 2006")
	return ui.homeSurface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(14).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, title, 15, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Start)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, weekdayCells(ui)...)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, calendarRows(func(day int) layout.Widget {
						return func(gtx layout.Context) layout.Dimensions {
							if day <= 0 || day > days {
								return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(34))}
							}
							date := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, now.Location())
							name := date.Format("2006-01-02") + ".md"
							click := ui.dayClickable(name)
							for click.Clicked(gtx) {
								ui.openNote(name)
							}
							return ui.dayTile(gtx, click, day, day == now.Day())
						}
					}, offset)...)
				}),
			)
		})
	})
}

func (ui *journalUI) recentNotes(gtx layout.Context) layout.Dimensions {
	return ui.homeSurface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			notes := listMarkdownNotes(ui.cfg.Session.VaultPath, 8)
			if len(notes) == 0 {
				return layout.Inset{Left: 4, Right: 4, Top: 8, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "No markdown notes yet.", 14, mutedColor(ui.cfg), text.Start)
				})
			}
			widgets := make([]layout.FlexChild, 0, len(notes))
			for _, note := range notes {
				name := note
				widgets = append(widgets, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					click := ui.noteClickable(name)
					for click.Clicked(gtx) {
						ui.openNote(name)
					}
					return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.noteTile(gtx, click, name)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, widgets...)
		})
	})
}

func (ui *journalUI) dayTile(gtx layout.Context, click *widget.Clickable, day int, today bool) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, gtx.Dp(34))
		bg := parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xf7, G: 0xf7, B: 0xf5, A: 0xff})
		if today {
			bg = parseColor(ui.cfg.Theme.Accent, color.NRGBA{R: 0xdc, G: 0xe6, B: 0xff, A: 0xff})
		} else if click.Hovered() {
			bg = blend(bg, parseColor(ui.cfg.Theme.Accent, color.NRGBA{A: 0xff}))
		}
		paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(9)).Op(gtx.Ops))
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, strconv.Itoa(day), 13, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Middle)
		})
	})
}

func (ui *journalUI) noteTile(gtx layout.Context, click *widget.Clickable, name string) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		gtx.Constraints.Min.Y = gtx.Dp(42)
		bg := parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xf7, G: 0xf7, B: 0xf5, A: 0xff})
		if click.Hovered() {
			bg = blend(bg, parseColor(ui.cfg.Theme.Accent, color.NRGBA{A: 0xff}))
		}
		paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(12)).Op(gtx.Ops))
		return layout.Inset{Left: 14, Right: 14, Top: 10, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, strings.TrimSuffix(name, ".md"), 14, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Start)
		})
	})
}

func (ui *journalUI) homeSurface(gtx layout.Context, content layout.Widget) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return widget.Border{
		Color:        parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}),
		CornerRadius: 14,
		Width:        1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				bg := blend(parseColor(ui.cfg.Theme.Background, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}), parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xf7, G: 0xf7, B: 0xf5, A: 0xff}))
				paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(14)).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			content,
		)
	})
}

func (ui *journalUI) noteClickable(name string) *widget.Clickable {
	if ui.noteButtons[name] == nil {
		ui.noteButtons[name] = &widget.Clickable{}
	}
	return ui.noteButtons[name]
}

func (ui *journalUI) dayClickable(name string) *widget.Clickable {
	if ui.dayButtons[name] == nil {
		ui.dayButtons[name] = &widget.Clickable{}
	}
	return ui.dayButtons[name]
}

type previewBlock struct {
	Text       string
	Size       unit.Sp
	Color      color.NRGBA
	Font       font.Font
	Segments   []previewSegment
	ImagePath  string
	ImageWidth int
	Table      [][]string
}

type previewSegment struct {
	Text string
	Font font.Font
}

func (block previewBlock) segmentWidgets(ui *journalUI) []layout.FlexChild {
	children := make([]layout.FlexChild, 0, len(block.Segments))
	for _, segment := range block.Segments {
		seg := segment
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.styledLabel(gtx, seg.Text, block.Size, block.Color, text.Start, seg.Font)
		}))
	}
	return children
}

func markdownBlocks(src string, cfg Config, baseDir string) []previewBlock {
	var blocks []previewBlock
	textColor := parseColor(cfg.Theme.Text, color.NRGBA{R: 0x20, G: 0x21, B: 0x24, A: 0xff})
	muted := mutedColor(cfg)
	baseFont := font.Font{}
	inCode := false
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		line = strings.TrimRight(line, "\r")
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inCode = !inCode
			continue
		}
		if trim == "" {
			blocks = append(blocks, previewBlock{Text: " ", Size: 8, Color: textColor})
			continue
		}
		if !inCode && isMarkdownTableRow(trim) {
			rows := [][]string{}
			for i < len(lines) {
				rowTrim := strings.TrimSpace(strings.TrimRight(lines[i], "\r"))
				if !isMarkdownTableRow(rowTrim) {
					break
				}
				if !isMarkdownTableSeparator(rowTrim) {
					rows = append(rows, parseTableCells(rowTrim))
				}
				i++
			}
			i--
			if len(rows) > 0 {
				blocks = append(blocks, previewBlock{Table: rows, Size: 14, Color: textColor, Font: baseFont})
			}
			continue
		}
		if alt, path, width, ok := parseMarkdownImage(trim); ok {
			blocks = append(blocks, imageBlock(alt, path, width, baseDir, muted, baseFont))
			continue
		}
		if path, width, ok := parseBareImagePath(trim); ok {
			blocks = append(blocks, imageBlock("", path, width, baseDir, muted, baseFont))
			continue
		}
		size := unit.Sp(16)
		col := textColor
		fnt := baseFont
		switch {
		case inCode:
			trim = "    " + line
			col = muted
			fnt.Typeface = "monospace"
		case isSectionDivider(trim):
			trim = "━━━━━━━━━━━━━━━━"
			col = muted
		case strings.HasPrefix(trim, "### "):
			trim = strings.TrimSpace(strings.TrimPrefix(trim, "### "))
			size = 20
			fnt.Weight = font.Bold
		case strings.HasPrefix(trim, "## "):
			trim = strings.TrimSpace(strings.TrimPrefix(trim, "## "))
			size = 24
			fnt.Weight = font.Bold
		case strings.HasPrefix(trim, "# "):
			trim = strings.TrimSpace(strings.TrimPrefix(trim, "# "))
			size = 30
			fnt.Weight = font.Bold
		case strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* "):
			trim = "- " + strings.TrimSpace(trim[2:])
		case orderedListItem(trim):
			dot := strings.Index(trim, ".")
			trim = strings.TrimSpace(trim[:dot+1]) + " " + strings.TrimSpace(trim[dot+1:])
		case strings.HasPrefix(trim, ">"):
			trim = strings.TrimSpace(strings.TrimPrefix(trim, ">"))
			trim = "| " + trim
			col = muted
			fnt.Style = font.Italic
		}
		segments := parseInlineSegments(trim, fnt)
		if len(segments) == 1 && segments[0].Font == fnt {
			blocks = append(blocks, previewBlock{Text: segments[0].Text, Size: size, Color: col, Font: fnt})
		} else {
			blocks = append(blocks, previewBlock{Size: size, Color: col, Font: fnt, Segments: segments})
		}
	}
	if len(blocks) == 0 {
		blocks = append(blocks, previewBlock{Text: "No preview content", Size: 16, Color: muted, Font: baseFont})
	}
	return blocks
}

func imageBlock(alt, path string, width int, baseDir string, muted color.NRGBA, baseFont font.Font) previewBlock {
	resolved := resolveImagePath(path, baseDir)
	label := alt
	if label == "" {
		label = filepath.Base(path)
	}
	return previewBlock{
		Text:       "Image: " + label,
		Size:       14,
		Color:      muted,
		Font:       baseFont,
		ImagePath:  resolved,
		ImageWidth: width,
	}
}

func parseMarkdownImage(s string) (alt, path string, width int, ok bool) {
	if !strings.HasPrefix(s, "![") {
		return "", "", 0, false
	}
	closeAlt := strings.Index(s, "](")
	closePath := strings.Index(s[closeAlt+2:], ")")
	if closeAlt < 2 || closePath < 0 {
		return "", "", 0, false
	}
	closePath += closeAlt + 2
	alt = s[2:closeAlt]
	path = strings.TrimSpace(s[closeAlt+2 : closePath])
	if path == "" || strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", "", 0, false
	}
	width = parseImageWidthSuffix(strings.TrimSpace(s[closePath+1:]))
	return alt, path, width, true
}

func parseBareImagePath(s string) (path string, width int, ok bool) {
	path = strings.TrimSpace(s)
	if suffixStart := strings.LastIndex(path, "{"); suffixStart >= 0 && strings.HasSuffix(path, "}") {
		width = parseImageWidthSuffix(path[suffixStart:])
		path = strings.TrimSpace(path[:suffixStart])
	}
	if !isImageFilePath(path) {
		return "", 0, false
	}
	return path, width, true
}

func resolveImagePath(path, baseDir string) string {
	path = strings.Trim(path, ` "'`)
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

func isImageFilePath(s string) bool {
	s = strings.Trim(strings.TrimSpace(s), `"'`)
	switch strings.ToLower(filepath.Ext(s)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	}
	return false
}

func parseInlineSegments(s string, base font.Font) []previewSegment {
	var segments []previewSegment
	for s != "" {
		next := nextInlineMarker(s)
		if next < 0 {
			segments = append(segments, previewSegment{Text: s, Font: base})
			break
		}
		if next > 0 {
			segments = append(segments, previewSegment{Text: s[:next], Font: base})
			s = s[next:]
		}
		marker := inlineMarker(s)
		if marker == "" {
			segments = append(segments, previewSegment{Text: s[:1], Font: base})
			s = s[1:]
			continue
		}
		end := strings.Index(s[len(marker):], marker)
		if end < 0 {
			segments = append(segments, previewSegment{Text: marker, Font: base})
			s = s[len(marker):]
			continue
		}
		content := s[len(marker) : len(marker)+end]
		fnt := base
		switch marker {
		case "**", "__":
			fnt.Weight = font.Bold
		case "*", "_":
			fnt.Style = font.Italic
		case "`":
			fnt.Typeface = "monospace"
		}
		segments = append(segments, previewSegment{Text: content, Font: fnt})
		s = s[len(marker)+end+len(marker):]
	}
	if len(segments) == 0 {
		return []previewSegment{{Text: "", Font: base}}
	}
	return segments
}

func nextInlineMarker(s string) int {
	best := -1
	for _, marker := range []string{"**", "__", "*", "_", "`"} {
		if i := strings.Index(s, marker); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	return best
}

func inlineMarker(s string) string {
	for _, marker := range []string{"**", "__", "*", "_", "`"} {
		if strings.HasPrefix(s, marker) {
			return marker
		}
	}
	return ""
}

func orderedListItem(s string) bool {
	dot := strings.Index(s, ".")
	if dot <= 0 || dot > 3 || len(s) <= dot+1 || s[dot+1] != ' ' {
		return false
	}
	for _, r := range s[:dot] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isSectionDivider(s string) bool {
	s = strings.TrimSpace(s)
	return s == "---" || s == "***" || s == "___"
}

func isMarkdownTableRow(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "|") && strings.HasSuffix(s, "|") && strings.Count(s, "|") >= 2
}

func isMarkdownTableSeparator(s string) bool {
	if !isMarkdownTableRow(s) {
		return false
	}
	for _, r := range strings.Trim(s, "| ") {
		if r != '-' && r != ':' && r != '|' && r != ' ' {
			return false
		}
	}
	return true
}

func parseTableCells(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.Trim(row, "|")
	parts := strings.Split(row, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func tableColumnCount(rows [][]string) int {
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	return cols
}

func tableRowWidgets(ui *journalUI, rows [][]string, cols int) []layout.FlexChild {
	children := make([]layout.FlexChild, 0, len(rows))
	for rowIndex, row := range rows {
		r := row
		idx := rowIndex
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			cells := make([]layout.FlexChild, 0, cols)
			for col := 0; col < cols; col++ {
				text := ""
				if col < len(r) {
					text = r[col]
				}
				cellText := text
				cells = append(cells, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.tableCell(gtx, cellText, idx == 0)
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, cells...)
		}))
	}
	return children
}

func (ui *journalUI) tableCell(gtx layout.Context, content string, header bool) layout.Dimensions {
	bg := parseColor(ui.cfg.Theme.Background, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	if header {
		bg = parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xf7, G: 0xf7, B: 0xf5, A: 0xff})
	}
	return widget.Border{
		Color: parseColor(ui.cfg.Theme.Chrome, color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}),
		Width: 1,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Min}.Op())
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: 10, Right: 10, Top: 8, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					fnt := font.Font{}
					if header {
						fnt.Weight = font.Bold
					}
					return ui.styledLabel(gtx, content, 14, parseColor(ui.cfg.Theme.Text, color.NRGBA{A: 0xff}), text.Start, fnt)
				})
			},
		)
	})
}

func spacer(w, h int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(w)), gtx.Dp(unit.Dp(h)))}
	}
}

func parseColor(s string, fallback color.NRGBA) color.NRGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return fallback
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return fallback
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

func rgbLabel(s string) string {
	col := parseColor(s, color.NRGBA{})
	if col.A == 0 {
		return "RGB --"
	}
	return fmt.Sprintf("RGB %d %d %d", col.R, col.G, col.B)
}

func findCountLabel(haystack, needle string) string {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return "0"
	}
	return strconv.Itoa(strings.Count(strings.ToLower(haystack), strings.ToLower(needle)))
}

func findPositionLabel(haystack, needle string, current int) string {
	matches := findMatchRanges(haystack, needle)
	if len(matches) == 0 {
		return "0/0"
	}
	if current < 0 {
		current = 0
	}
	if current >= len(matches) {
		current = len(matches) - 1
	}
	return fmt.Sprintf("%d/%d", current+1, len(matches))
}

type matchRange struct {
	start int
	end   int
}

func findMatchRanges(haystack, needle string) []matchRange {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return nil
	}
	lowerHaystack := strings.ToLower(haystack)
	lowerNeedle := strings.ToLower(needle)
	var matches []matchRange
	searchFrom := 0
	for {
		index := strings.Index(lowerHaystack[searchFrom:], lowerNeedle)
		if index < 0 {
			break
		}
		byteStart := searchFrom + index
		byteEnd := byteStart + len(lowerNeedle)
		matches = append(matches, matchRange{
			start: runeOffset(haystack, byteStart),
			end:   runeOffset(haystack, byteEnd),
		})
		searchFrom = byteEnd
	}
	return matches
}

func runeOffset(s string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(s) {
		return len([]rune(s))
	}
	return len([]rune(s[:byteOffset]))
}

func matchIndexAtOrAfter(matches []matchRange, runeOffset int) int {
	for i, match := range matches {
		if match.start >= runeOffset {
			return i
		}
	}
	return len(matches)
}

func exactMatchIndex(matches []matchRange, start, end int) int {
	for i, match := range matches {
		if match.start == start && match.end == end {
			return i
		}
	}
	return -1
}

func replaceAllCaseInsensitive(haystack, needle, replacement string) (string, int) {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return haystack, 0
	}
	lowerHaystack := strings.ToLower(haystack)
	lowerNeedle := strings.ToLower(needle)
	var b strings.Builder
	searchFrom := 0
	count := 0
	for {
		index := strings.Index(lowerHaystack[searchFrom:], lowerNeedle)
		if index < 0 {
			b.WriteString(haystack[searchFrom:])
			break
		}
		byteStart := searchFrom + index
		byteEnd := byteStart + len(lowerNeedle)
		b.WriteString(haystack[searchFrom:byteStart])
		b.WriteString(replacement)
		searchFrom = byteEnd
		count++
	}
	return b.String(), count
}

func parseImageWidthSuffix(s string) int {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return 0
	}
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "{"), "}"))
	if !strings.HasPrefix(s, "w=") {
		return 0
	}
	width, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(s, "w=")))
	if err != nil || width < 80 {
		return 0
	}
	return minInt(width, 1600)
}

func extractImagePath(s string) string {
	if _, path, _, ok := parseMarkdownImage(strings.TrimSpace(s)); ok {
		return path
	}
	path, _, ok := parseBareImagePath(s)
	if ok {
		return path
	}
	return strings.TrimSpace(s)
}

func normalizeColor(s, fallback string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		s = "#" + s
	}
	if parseColor(s, color.NRGBA{}).A == 0 {
		return fallback
	}
	return strings.ToUpper(s)
}

func mutedColor(cfg Config) color.NRGBA {
	return parseColor(cfg.Theme.Muted, color.NRGBA{R: 0x7b, G: 0x81, B: 0x88, A: 0xff})
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func absFloat(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func blend(a, b color.NRGBA) color.NRGBA {
	return color.NRGBA{
		R: uint8((uint16(a.R)*2 + uint16(b.R)) / 3),
		G: uint8((uint16(a.G)*2 + uint16(b.G)) / 3),
		B: uint8((uint16(a.B)*2 + uint16(b.B)) / 3),
		A: 0xff,
	}
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func calendarRows(dayWidget func(day int) layout.Widget, offset int) []layout.FlexChild {
	var rows []layout.FlexChild
	day := 1 - offset
	for row := 0; row < 6; row++ {
		start := day
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, 7)
			for col := 0; col < 7; col++ {
				widget := dayWidget(start + col)
				children = append(children, layout.Flexed(1, widget))
			}
			return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
			})
		}))
		day += 7
	}
	return rows
}

func weekdayCells(ui *journalUI) []layout.FlexChild {
	names := []string{"S", "M", "T", "W", "T", "F", "S"}
	children := make([]layout.FlexChild, 0, len(names))
	for _, name := range names {
		label := name
		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, label, 11, mutedColor(ui.cfg), text.Middle)
		}))
	}
	return children
}

func looksLikeImagePath(s string) bool {
	s = strings.Trim(strings.TrimSpace(s), `"'`)
	if s == "" || strings.ContainsAny(s, "\n\r") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(s))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return true
	}
	return strings.Contains(s, "/") || strings.Contains(s, "\\")
}

func listMarkdownNotes(vault string, limit int) []string {
	entries, err := os.ReadDir(vault)
	if err != nil {
		return nil
	}
	type noteInfo struct {
		name string
		mod  time.Time
	}
	var notes []noteInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		notes = append(notes, noteInfo{name: entry.Name(), mod: info.ModTime()})
	}
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].mod.After(notes[j].mod)
	})
	if len(notes) > limit {
		notes = notes[:limit]
	}
	out := make([]string, len(notes))
	for i, note := range notes {
		out[i] = note.name
	}
	return out
}

func iconFromData(data []byte) *widget.Icon {
	icon, err := widget.NewIcon(data)
	if err != nil {
		return nil
	}
	return icon
}

func nativeFolderDialog(title, current string) (string, bool) {
	path, ok := runZenityDialog("--file-selection", "--directory", "--title="+title, "--filename="+dialogStartPath(current))
	if !ok {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return path, true
}

func nativeFileDialog(title, current string, filters []string) (string, bool) {
	args := []string{"--file-selection", "--title=" + title, "--filename=" + dialogStartPath(current)}
	for _, filter := range filters {
		args = append(args, "--file-filter="+filter)
	}
	path, ok := runZenityDialog(args...)
	if !ok {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

func runZenityDialog(args ...string) (string, bool) {
	if _, err := exec.LookPath("zenity"); err != nil {
		return "", false
	}
	out, err := exec.Command("zenity", args...).Output()
	if err != nil {
		return "", false
	}
	path := strings.TrimSpace(string(out))
	return path, path != ""
}

func dialogStartPath(current string) string {
	current = strings.TrimSpace(current)
	if current != "" {
		if info, err := os.Stat(current); err == nil && info.IsDir() {
			return current + string(os.PathSeparator)
		}
		return current
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + string(os.PathSeparator)
}

func userFacingError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, os.ErrNotExist) {
		return "File not found"
	}
	return fmt.Sprintf("%v", err)
}
