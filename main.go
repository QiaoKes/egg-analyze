package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"path/filepath"
	"strings"
	"time"

	"egg-analyze/internal/analyzer"
	"egg-analyze/internal/capture"
	"egg-analyze/internal/config"
	"egg-analyze/internal/hotkey"
	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type uiState struct {
	app         fyne.App
	window      fyne.Window
	statusLabel *widget.Label
	sourceLabel *widget.Label
	logLabel    *widget.Label
	resultsBox  *fyne.Container
	previewView *canvas.Image

	config          *config.Manager
	hotkeys         *hotkey.Manager
	ocr             ocr.Recognizer
	engine          *rocom.Engine
	analyzeSvc      *analyzer.Service
	mainVisible     bool
	captureInFlight bool

	currentImage image.Image
	quitting     bool
}

func main() {
	cfgMgr, err := config.Load()
	if err != nil {
		panic(err)
	}

	application := app.NewWithID("egg-analyze")
	application.Settings().SetTheme(newContrastTheme())
	if icon, iconErr := buildTrayResource(); iconErr == nil {
		application.SetIcon(icon)
	}
	window := application.NewWindow("洛克王国精灵蛋分析")

	state := &uiState{
		app:     application,
		window:  window,
		config:  cfgMgr,
		hotkeys: hotkey.NewManager(),
		ocr:     ocr.New(),
	}

	state.buildUI()
	state.setupTray()
	state.attachCloseBehavior()

	if err := state.reloadDataset(); err != nil {
		state.setStatus("数据加载失败")
		state.log(fmt.Sprintf("数据加载失败: %v", err))
	} else {
		state.setStatus("数据已加载")
	}

	if err := state.registerHotkey(state.config.Get().Hotkey); err != nil {
		state.log(fmt.Sprintf("快捷键注册失败: %v", err))
	}

	window.Resize(fyne.NewSize(1360, 860))
	state.mainVisible = true
	window.ShowAndRun()
	state.hotkeys.Close()
}

func (s *uiState) buildUI() {
	s.statusLabel = widget.NewLabel("准备就绪")
	s.sourceLabel = widget.NewLabel("数据源：未加载")

	s.logLabel = widget.NewLabel("启动完成，等待操作。")
	s.logLabel.Wrapping = fyne.TextWrapWord
	s.logLabel.TextStyle = fyne.TextStyle{Monospace: true}

	s.resultsBox = container.NewVBox(s.buildEmptyResults())

	s.previewView = canvas.NewImageFromImage(buildPlaceholderImage())
	s.previewView.FillMode = canvas.ImageFillContain
	s.previewView.SetMinSize(fyne.NewSize(720, 520))

	toolbar := container.NewHBox(
		widget.NewButton("截图分析", func() {
			s.beginCaptureSelection()
		}),
		widget.NewButton("打开图片", s.openImageDialog),
		widget.NewButton("刷新数据", func() {
			s.run("刷新数据", s.reloadDataset)
		}),
		widget.NewButton("保存截图", s.saveCurrentCapture),
		widget.NewButton("设置", s.openSettingsDialog),
	)

	statusStrip := container.NewGridWithColumns(2,
		widget.NewCard("当前状态", "", s.statusLabel),
		widget.NewCard("数据源", "", s.sourceLabel),
	)

	previewPanel := widget.NewCard("截图预览", "用于确认 OCR 是否读对区域和数字", container.NewPadded(container.NewCenter(s.previewView)))
	resultsPanel := widget.NewCard("分析结果", "按蛋分组查看 size / weight 与候选概率", container.NewScroll(container.NewPadded(s.resultsBox)))

	mainSplit := container.NewHSplit(previewPanel, resultsPanel)
	mainSplit.Offset = 0.54

	logAccordion := widget.NewAccordion(
		widget.NewAccordionItem("运行日志", container.NewScroll(container.NewPadded(s.logLabel))),
	)
	logAccordion.CloseAll()

	content := container.NewBorder(
		container.NewVBox(toolbar, statusStrip),
		logAccordion,
		nil,
		nil,
		mainSplit,
	)

	s.window.SetContent(container.NewPadded(content))
}

func (s *uiState) setupTray() {
	desk, ok := s.app.(desktop.App)
	if !ok {
		s.log("当前平台不支持系统托盘")
		return
	}

	icon, err := buildTrayResource()
	if err != nil {
		s.log(fmt.Sprintf("托盘图标初始化失败: %v", err))
	} else {
		s.app.SetIcon(icon)
	}
	desk.SetSystemTrayMenu(fyne.NewMenu("洛克王国精灵蛋分析",
		fyne.NewMenuItem("显示窗口", s.showWindow),
		fyne.NewMenuItem("截图分析", func() {
			s.beginCaptureSelection()
		}),
		fyne.NewMenuItem("刷新数据", func() {
			s.run("刷新数据", s.reloadDataset)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("退出", s.quit),
	))
}

func (s *uiState) attachCloseBehavior() {
	s.window.SetCloseIntercept(func() {
		if s.quitting {
			s.window.SetCloseIntercept(nil)
			s.window.Close()
			return
		}
		s.window.Hide()
		s.mainVisible = false
		s.log("窗口已隐藏到托盘")
	})
}

func (s *uiState) reloadDataset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dataset, info, err := rocom.LoadDataset(ctx, s.config.Get().DatasetURL)
	if err != nil {
		return err
	}

	engine, err := rocom.NewEngine(dataset)
	if err != nil {
		return err
	}

	s.engine = engine
	s.analyzeSvc = analyzer.NewService(s.ocr, engine, s.config.Get().TopCandidate)
	source := "远程"
	if info.FromCache {
		source = "本地缓存"
	}

	s.sourceLabel.SetText(fmt.Sprintf("数据源：%s | 记录 %d | 更新时间 %s", source, info.RecordCount, info.UpdatedAt.Format("2006-01-02 15:04:05")))
	s.log(fmt.Sprintf("数据已加载，来源=%s，记录=%d", info.URL, info.RecordCount))
	return nil
}

func (s *uiState) handleImage(img image.Image, source string) error {
	if s.analyzeSvc == nil {
		return fmt.Errorf("数据尚未加载，请先点击“刷新网站数据”")
	}

	results, rawLines, err := s.analyzeSvc.AnalyzeImage(context.Background(), img)
	if err != nil {
		return fmt.Errorf("%w；OCR原始行数=%d", err, len(rawLines))
	}

	s.currentImage = img
	capturePath, _ := config.LastCapturePath()
	if capturePath != "" {
		_ = capture.SavePNG(capturePath, img)
	}

	s.updatePreview(img)
	s.renderResults(results)
	s.log(fmt.Sprintf("%s 完成，识别到 %d 组尺寸/重量", source, len(results)))
	return nil
}

func (s *uiState) renderResults(results []analyzer.Result) {
	objects := make([]fyne.CanvasObject, 0, len(results))
	for idx, result := range results {
		candidates := make([]fyne.CanvasObject, 0, len(result.Candidates)+1)
		metrics := container.NewGridWithColumns(2,
			buildMetricRow("Size", fmt.Sprintf("%.3f", result.Measurement.Size)),
			buildMetricRow("Weight", fmt.Sprintf("%.3f", result.Measurement.Weight)),
		)
		candidates = append(candidates, metrics)
		for candidateIndex, candidate := range result.Candidates {
			title := fmt.Sprintf("%d. %s (%s)", candidateIndex+1, candidate.Pet, candidate.PetID)
			meta := fmt.Sprintf("概率 %.2f%% | 参考尺寸 %s | 参考重量 %s | %s", candidate.Probability, candidate.EggDiameter, candidate.EggWeight, candidate.MatchType)
			line := widget.NewLabel(title + "\n" + meta)
			line.Wrapping = fyne.TextWrapWord
			if candidateIndex == 0 {
				line.Importance = widget.HighImportance
			}
			candidates = append(candidates, line)
		}
		objects = append(objects, widget.NewCard(
			fmt.Sprintf("蛋 %d", idx+1),
			"",
			container.NewVBox(candidates...),
		))
	}
	if len(objects) == 0 {
		objects = append(objects, s.buildEmptyResults())
	}
	s.resultsBox.Objects = objects
	s.resultsBox.Refresh()
}

func buildMetricRow(name, value string) fyne.CanvasObject {
	label := widget.NewLabel(name)
	label.Importance = widget.MediumImportance
	number := widget.NewLabelWithStyle(value, fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	return container.NewPadded(container.NewBorder(nil, nil, label, nil, number))
}

func (s *uiState) updatePreview(img image.Image) {
	s.previewView.Image = img
	s.previewView.Refresh()
}

func (s *uiState) openImageDialog() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			s.log(fmt.Sprintf("打开图片对话框失败: %v", err))
			dialog.ShowError(err, s.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		img, _, decodeErr := image.Decode(reader)
		if decodeErr != nil {
			s.log(fmt.Sprintf("解析图片失败: %v", decodeErr))
			dialog.ShowError(decodeErr, s.window)
			return
		}

		path := reader.URI().Path()
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			cfg := s.config.Get()
			cfg.LastImageDir = dir
			if saveErr := s.config.Save(cfg); saveErr != nil {
				s.log(fmt.Sprintf("保存图片目录失败: %v", saveErr))
			}
		}
		s.setStatus("图片分析处理中")
		if analyzeErr := s.handleImage(img, path); analyzeErr != nil {
			s.setStatus("图片分析失败")
			s.log(fmt.Sprintf("图片分析失败: %v", analyzeErr))
			dialog.ShowError(analyzeErr, s.window)
			return
		}
		s.setStatus("图片分析完成")
	}, s.window)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg"}))
	if lastDir := strings.TrimSpace(s.config.Get().LastImageDir); lastDir != "" {
		if uri := storage.NewFileURI(lastDir); uri != nil {
			if lister, err := storage.ListerForURI(uri); err == nil {
				fileDialog.SetLocation(lister)
			} else {
				s.log(fmt.Sprintf("恢复图片目录失败: %v", err))
			}
		}
	}
	fileDialog.Show()
}

func (s *uiState) saveCurrentCapture() {
	if s.currentImage == nil {
		s.log("当前没有可保存的截图")
		return
	}

	fileDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			s.log(fmt.Sprintf("保存对话框失败: %v", err))
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()

		if encodeErr := png.Encode(writer, s.currentImage); encodeErr != nil {
			s.log(fmt.Sprintf("保存截图失败: %v", encodeErr))
			return
		}
		s.log(fmt.Sprintf("截图已保存到 %s", writer.URI().Path()))
	}, s.window)
	fileDialog.SetFileName("egg-capture.png")
	fileDialog.Show()
}

func (s *uiState) beginCaptureSelection() {
	if s.captureInFlight {
		s.log("截图流程正在进行中")
		return
	}
	if s.analyzeSvc == nil {
		err := fmt.Errorf("数据尚未加载，请先点击“刷新网站数据”")
		s.setStatus("截图分析失败")
		s.log(err.Error())
		dialog.ShowError(err, s.window)
		return
	}

	restoreWindow := s.mainVisible
	if restoreWindow {
		s.window.Hide()
		s.mainVisible = false
	}

	s.captureInFlight = true
	s.setStatus("等待 Flameshot 框选")
	go func() {
		time.Sleep(140 * time.Millisecond)

		img, err := capture.CaptureWithFlameshot(context.Background())
		s.captureInFlight = false
		switch {
		case err == nil:
			s.showWindow()
			s.run("截图分析", func() error {
				return s.handleImage(img, "Flameshot 截图")
			})
		case errors.Is(err, capture.ErrCancelled):
			s.setStatus("准备就绪")
			s.log("已取消截图")
			if restoreWindow {
				s.showWindow()
			}
		default:
			s.showWindow()
			s.setStatus("截图分析失败")
			s.log(fmt.Sprintf("Flameshot 截图失败: %v", err))
			dialog.ShowError(err, s.window)
		}
	}()
}

func (s *uiState) saveHotkey(hotkeyText string) {
	newCfg := s.config.Get()
	newCfg.Hotkey = strings.TrimSpace(hotkeyText)
	if err := s.config.Save(newCfg); err != nil {
		s.log(fmt.Sprintf("保存快捷键失败: %v", err))
		dialog.ShowError(err, s.window)
		return
	}
	if err := s.registerHotkey(newCfg.Hotkey); err != nil {
		s.log(fmt.Sprintf("重新注册快捷键失败: %v", err))
		dialog.ShowError(err, s.window)
		return
	}
	s.log("快捷键已更新")
}

func (s *uiState) registerHotkey(hotkeyText string) error {
	return s.hotkeys.Register(hotkeyText, func() {
		s.beginCaptureSelection()
	})
}

func (s *uiState) run(name string, fn func() error) {
	s.setStatus(name + "处理中")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.setStatus(name + "失败")
				s.log(fmt.Sprintf("%s失败: %v", name, r))
			}
		}()

		err := fn()
		if err != nil {
			s.setStatus(name + "失败")
			s.log(fmt.Sprintf("%s失败: %v", name, err))
			return
		}
		s.setStatus(name + "完成")
	}()
}

func (s *uiState) log(message string) {
	current := strings.TrimSpace(s.logLabel.Text)
	timestamped := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message)
	if current == "" {
		s.logLabel.SetText(timestamped)
		return
	}
	s.logLabel.SetText(current + "\n" + timestamped)
}

func (s *uiState) setStatus(message string) {
	s.statusLabel.SetText(message)
}

func (s *uiState) showWindow() {
	s.window.Show()
	s.mainVisible = true
	s.window.RequestFocus()
}

func (s *uiState) openSettingsDialog() {
	cfg := s.config.Get()
	currentHotkey := widget.NewLabel(hotkey.NormalizeShortcut(cfg.Hotkey))
	currentHotkey.Importance = widget.HighImportance
	currentHotkey.Wrapping = fyne.TextWrapWord

	helper := widget.NewLabel("点击“录入快捷键”后，直接按下新的组合键。录入成功会立即保存。")
	helper.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		widget.NewCard("全局热键", "", container.NewVBox(
			currentHotkey,
			helper,
			widget.NewButton("录入快捷键", func() {
				s.openHotkeyCaptureWindow(currentHotkey)
			}),
		)),
	)

	dialog.ShowCustom("设置", "关闭", content, s.window)
}

func (s *uiState) openHotkeyCaptureWindow(currentHotkey *widget.Label) {
	win := s.app.NewWindow("录入快捷键")
	win.SetFixedSize(true)

	captureBox := newHotkeyCaptureWidget(s.config.Get().Hotkey, func(value string) {
		currentHotkey.SetText(hotkey.NormalizeShortcut(value))
		s.saveHotkey(value)
		win.Close()
	}, func() {
		win.Close()
	})

	content := container.NewVBox(
		widget.NewLabel("按下你要使用的快捷键组合。"),
		widget.NewLabel("录入会自动结束；按 Esc 取消。"),
		captureBox,
	)
	win.SetContent(container.NewPadded(content))
	win.Resize(fyne.NewSize(420, 220))
	win.Show()
	win.RequestFocus()
	if canvas := win.Canvas(); canvas != nil {
		canvas.Focus(captureBox)
	}
}

func (s *uiState) buildEmptyResults() fyne.CanvasObject {
	label := widget.NewLabel("暂无分析结果。\n先截图或导入图片，结果会按蛋分组显示。")
	label.Wrapping = fyne.TextWrapWord
	return widget.NewCard("分析结果", "", label)
}

func (s *uiState) quit() {
	s.quitting = true
	s.hotkeys.Close()
	s.window.SetCloseIntercept(nil)
	s.window.Close()
}

func buildTrayResource() (fyne.Resource, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, buildTrayImage()); err != nil {
		return nil, err
	}
	return fyne.NewStaticResource("tray.png", buffer.Bytes()), nil
}

func buildTrayImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	bg := color.RGBA{R: 33, G: 37, B: 41, A: 255}
	fg := color.RGBA{R: 255, G: 214, B: 10, A: 255}
	hl := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, bg)
		}
	}

	for y := 6; y < 26; y++ {
		for x := 8; x < 24; x++ {
			if (x-16)*(x-16)+(y-16)*(y-16) <= 90 {
				img.Set(x, y, fg)
			}
		}
	}

	for y := 12; y < 20; y++ {
		img.Set(15, y, hl)
	}
	for x := 14; x < 18; x++ {
		img.Set(x, 20, hl)
	}

	return img
}

func buildPlaceholderImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 520, 320))
	for y := 0; y < 320; y++ {
		for x := 0; x < 520; x++ {
			img.Set(x, y, color.RGBA{R: 244, G: 246, B: 248, A: 255})
		}
	}
	return img
}

type contrastTheme struct {
	base fyne.Theme
}

func newContrastTheme() fyne.Theme {
	return &contrastTheme{base: theme.LightTheme()}
}

func (t *contrastTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0xF4, G: 0xF1, B: 0xE8, A: 0xFF}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0xFF, G: 0xFD, B: 0xF8, A: 0xFF}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0xC3, G: 0xB7, B: 0x9C, A: 0xFF}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0xD8, G: 0xC2, B: 0x8C, A: 0xFF}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0x15, G: 0x18, B: 0x1C, A: 0xFF}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x4C, G: 0x53, B: 0x5D, A: 0xFF}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x56, G: 0x5F, B: 0x6B, A: 0xFF}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x88, G: 0x7A, B: 0x63, A: 0xCC}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0xA8, G: 0x59, B: 0x27, A: 0xFF}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0xA8, G: 0x59, B: 0x27, A: 0x22}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0xA8, G: 0x59, B: 0x27, A: 0x44}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0xD5, G: 0xCC, B: 0xB7, A: 0xFF}
	}
	return t.base.Color(name, variant)
}

func (t *contrastTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *contrastTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *contrastTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.base.Size(name)
}
