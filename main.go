package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"egg-analyze/internal/analyzer"
	"egg-analyze/internal/atlas"
	"egg-analyze/internal/capture"
	"egg-analyze/internal/config"
	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	sqdialog "github.com/sqweek/dialog"
)

type uiState struct {
	app          fyne.App
	bubbleWindow fyne.Window
	panelWindow  fyne.Window
	resultWindow fyne.Window

	panelStatus *widget.Label
	panelSource *widget.Label
	resultList  *fyne.Container

	config *config.Manager
	ocr    ocr.Recognizer
	atlas  *atlas.Service

	engine       *rocom.Engine
	analyzeSvc   *analyzer.Service
	results      []analyzer.Result
	currentImage image.Image

	captureInFlight bool
	panelVisible    bool
	resultVisible   bool
	bubblePrimed    bool
	bubbleRestored  bool
	bubbleAnchor    nativeWindowFrame
	bubbleAnchorSet bool
	quitting        bool
	cleanupOnce     sync.Once
}

func main() {
	cfgMgr, err := config.Load()
	if err != nil {
		panic(err)
	}

	application := app.New()
	application.Settings().SetTheme(newContrastTheme())

	state := &uiState{
		app:    application,
		config: cfgMgr,
		ocr:    ocr.New(),
		atlas:  atlas.New(),
	}

	state.bubbleWindow = state.newUtilityWindow("悬浮球")
	state.panelWindow = state.newUtilityWindow("洛克王国精灵蛋分析")
	state.resultWindow = state.newUtilityWindow("识别结果")
	state.bubbleWindow.SetMaster()

	state.buildBubbleUI()
	state.buildPanelUI()
	state.buildResultUI()
	state.configureNativeWindows()
	state.attachCloseBehavior()
	state.attachSignalHandler()

	application.Lifecycle().SetOnStarted(func() {
		state.showBubbleWindow()
		state.startupAsync()
	})

	state.bubbleWindow.Resize(fyne.NewSize(60, 60))
	state.panelWindow.Resize(fyne.NewSize(360, 412))
	state.resultWindow.Resize(fyne.NewSize(520, 680))
	application.Run()
	state.cleanup()
}

func (s *uiState) newUtilityWindow(title string) fyne.Window {
	if driver, ok := s.app.Driver().(desktop.Driver); ok {
		win := driver.CreateSplashWindow()
		win.SetTitle(title)
		win.SetFixedSize(true)
		win.SetPadded(false)
		return win
	}

	win := s.app.NewWindow(title)
	win.SetFixedSize(true)
	win.SetPadded(false)
	return win
}

func (s *uiState) buildBubbleUI() {
	if setupNativeBubbleWindow(s.bubbleWindow, s.togglePanelWindow, s.rememberBubbleAnchorSoon) {
		s.bubbleWindow.SetContent(canvas.NewRectangle(color.Transparent))
		return
	}
	bubble := newBubbleWidget(s.togglePanelWindow, func() {
		beginNativeWindowDrag(s.bubbleWindow)
	}, s.rememberBubbleAnchorSoon)
	s.bubbleWindow.SetContent(container.NewStack(canvas.NewRectangle(color.Transparent), container.NewCenter(bubble)))
}

func (s *uiState) buildPanelUI() {
	s.panelStatus = widget.NewLabel("准备就绪")
	s.panelStatus.Alignment = fyne.TextAlignCenter
	s.panelStatus.Wrapping = fyne.TextWrapWord
	s.panelStatus.TextStyle = fyne.TextStyle{Bold: true}

	s.panelSource = widget.NewLabel("数据源：启动中")
	s.panelSource.Alignment = fyne.TextAlignCenter
	s.panelSource.Wrapping = fyne.TextWrapWord

	title := newWindowDragLabel("洛克王国精灵蛋分析", s.panelWindow, fyne.TextAlignCenter)
	description := widget.NewLabel("点击主按钮开始框选截图")
	description.Alignment = fyne.TextAlignCenter
	description.Wrapping = fyne.TextWrapWord

	header := container.NewBorder(
		nil,
		nil,
		nil,
		newCompactGlassButton(theme.WindowMinimizeIcon(), s.hidePanelWindow),
		title,
	)
	header = container.NewStack(newWindowDragArea(s.panelWindow), header)

	captureButton := newPrimaryCaptureButton(func() {
		s.beginCaptureSelection()
	})

	actions := container.NewGridWithColumns(2,
		newSecondaryActionButton("打开图片", theme.FolderOpenIcon(), s.openImageDialog),
		newSecondaryActionButton("刷新数据", theme.ViewRefreshIcon(), func() {
			s.run("刷新数据", s.reloadDataset)
		}),
		newSecondaryActionButton("查看结果", theme.VisibilityIcon(), s.showResultWindow),
		newSecondaryActionButton("退出", theme.CancelIcon(), s.quit),
	)

	content := container.NewVBox(
		header,
		layoutSpacer(2),
		description,
		layoutSpacer(8),
		captureButton,
		layoutSpacer(6),
		s.panelStatus,
		layoutSpacer(2),
		s.panelSource,
		layoutSpacer(8),
		actions,
	)

	panel := buildSurfaceCard(content, color.NRGBA{R: 0xF7, G: 0xF9, B: 0xFD, A: 0x38}, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x58}, 34)
	s.panelWindow.SetContent(buildFloatingWindowContent(panel))
}

func (s *uiState) buildResultUI() {
	s.resultList = container.NewVBox(s.buildEmptyResultCard())
	scroll := container.NewScroll(container.NewPadded(s.resultList))
	scroll.SetMinSize(fyne.NewSize(480, 600))

	title := newWindowDragLabel("识别结果", s.resultWindow, fyne.TextAlignLeading)
	toolbar := container.NewBorder(nil, nil, nil, container.NewHBox(
		newSecondaryActionButton("再截一次", theme.MediaReplayIcon(), func() {
			s.beginCaptureSelection()
		}),
		newSecondaryActionButton("隐藏", theme.VisibilityOffIcon(), s.hideResultWindow),
	), title)
	toolbar = container.NewStack(newWindowDragArea(s.resultWindow), toolbar)

	body := container.NewBorder(
		container.NewPadded(toolbar),
		nil,
		nil,
		nil,
		scroll,
	)

	glass := buildSurfaceCard(body, color.NRGBA{R: 0xF8, G: 0xFA, B: 0xFE, A: 0x34}, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x54}, 36)
	s.resultWindow.SetContent(buildFloatingWindowContent(glass))
}

func buildSurfaceCard(content fyne.CanvasObject, fill color.Color, stroke color.Color, radius float32) fyne.CanvasObject {
	shadow := canvas.NewRectangle(color.NRGBA{R: 0x16, G: 0x20, B: 0x2E, A: 0x05})
	shadow.CornerRadius = radius
	shadow.Move(fyne.NewPos(0, 10))
	bg := canvas.NewRectangle(fill)
	bg.CornerRadius = radius
	bg.StrokeColor = stroke
	bg.StrokeWidth = 1
	shine := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x12})
	shine.CornerRadius = radius - 2
	shine.Move(fyne.NewPos(4, 4))
	return container.NewStack(shadow, bg, shine, container.NewPadded(content))
}

func buildFloatingWindowContent(card fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(
		canvas.NewRectangle(color.Transparent),
		card,
	)
}

func layoutSpacer(height float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(1, height))
	return spacer
}

func (s *uiState) configureNativeWindows() {
	configureNativeWindow(s.bubbleWindow, nativeWindowStyle{
		CornerRadius:        30,
		Floating:            true,
		Transparent:         true,
		MovableByBackground: true,
	})
	configureNativeWindow(s.panelWindow, nativeWindowStyle{
		CornerRadius:        34,
		Floating:            true,
		Transparent:         true,
		MovableByBackground: true,
	})
	configureNativeWindow(s.resultWindow, nativeWindowStyle{
		CornerRadius:        36,
		Floating:            true,
		Transparent:         true,
		MovableByBackground: true,
	})
}

func (s *uiState) attachCloseBehavior() {
	s.bubbleWindow.SetCloseIntercept(func() {
		if s.quitting {
			s.bubbleWindow.SetCloseIntercept(nil)
			s.bubbleWindow.Close()
			return
		}
		s.quit()
	})

	s.panelWindow.SetCloseIntercept(func() {
		if s.quitting {
			s.panelWindow.SetCloseIntercept(nil)
			s.panelWindow.Close()
			return
		}
		s.hidePanelWindow()
	})

	s.resultWindow.SetCloseIntercept(func() {
		if s.quitting {
			s.resultWindow.SetCloseIntercept(nil)
			s.resultWindow.Close()
			return
		}
		s.hideResultWindow()
	})
}

func (s *uiState) attachSignalHandler() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		s.quit()
	}()
}

func (s *uiState) startupAsync() {
	go func() {
		s.setStatus("启动中，正在加载数据")
		if err := s.reloadDataset(); err != nil {
			s.setStatus("数据加载失败")
			s.showError(err)
			return
		}
		s.setStatus("准备就绪")
	}()
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

	s.panelSource.SetText(fmt.Sprintf("数据源：%s | 记录 %d | 更新时间\n%s", source, info.RecordCount, info.UpdatedAt.Format("2006-01-02 15:04:05")))
	s.warmAtlasIndex()
	return nil
}

func (s *uiState) beginCaptureSelection() {
	if s.captureInFlight {
		return
	}
	if s.analyzeSvc == nil {
		err := fmt.Errorf("数据尚未加载，请先点击“刷新数据”")
		s.setStatus("截图分析失败")
		s.showError(err)
		return
	}

	panelWasVisible := s.panelVisible
	resultWasVisible := s.resultVisible
	s.captureInFlight = true
	s.setStatus("等待 Flameshot 框选")
	s.hidePanelWindow()
	s.hideResultWindow()
	s.hideBubbleWindow()

	go func() {
		time.Sleep(140 * time.Millisecond)

		img, err := capture.CaptureWithFlameshot(context.Background())
		s.captureInFlight = false
		s.showBubbleWindow()
		if panelWasVisible {
			s.showPanelWindow()
		}

		switch {
		case err == nil:
			s.run("截图分析", func() error {
				analyzeErr := s.handleImage(img, "Flameshot 截图")
				if analyzeErr == nil {
					s.showResultWindow()
				}
				return analyzeErr
			})
		case errors.Is(err, capture.ErrCancelled):
			s.setStatus("准备就绪")
			if resultWasVisible {
				s.showResultWindow()
			}
		default:
			s.setStatus("截图分析失败")
			if resultWasVisible {
				s.showResultWindow()
			}
			s.showError(err)
		}
	}()
}

func (s *uiState) openImageDialog() {
	panelWasVisible := s.panelVisible
	resultWasVisible := s.resultVisible
	s.hidePanelWindow()
	s.hideResultWindow()
	s.hideBubbleWindow()

	restoreLauncher := func() {
		switch {
		case panelWasVisible:
			s.showPanelWindow()
		case resultWasVisible:
			s.showResultWindow()
		default:
			s.showBubbleWindow()
		}
	}

	chooser := sqdialog.File().Title("选择图片").Filter("Image Files", "png", "jpg", "jpeg")
	if lastDir := strings.TrimSpace(s.config.Get().LastImageDir); lastDir != "" {
		chooser = chooser.SetStartDir(lastDir)
	}

	path, err := chooser.Load()
	if err != nil {
		if errors.Is(err, sqdialog.ErrCancelled) {
			restoreLauncher()
			return
		}
		restoreLauncher()
		s.showError(err)
		return
	}
	if strings.TrimSpace(path) == "" {
		restoreLauncher()
		return
	}

	file, err := os.Open(path)
	if err != nil {
		restoreLauncher()
		s.showError(err)
		return
	}
	defer file.Close()

	img, _, decodeErr := image.Decode(file)
	if decodeErr != nil {
		restoreLauncher()
		s.showError(decodeErr)
		return
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		cfg := s.config.Get()
		cfg.LastImageDir = dir
		_ = s.config.Save(cfg)
	}

	s.setStatus("图片分析处理中")
	if analyzeErr := s.handleImage(img, path); analyzeErr != nil {
		s.setStatus("图片分析失败")
		restoreLauncher()
		s.showError(analyzeErr)
		return
	}
	s.setStatus("图片分析完成")
	s.showResultWindow()
}

func (s *uiState) handleImage(img image.Image, source string) error {
	if s.analyzeSvc == nil {
		return fmt.Errorf("数据尚未加载，请先点击“刷新数据”")
	}

	results, rawLines, err := s.analyzeSvc.AnalyzeImage(context.Background(), img)
	if err != nil {
		return fmt.Errorf("%w；OCR原始行数=%d", err, len(rawLines))
	}

	s.currentImage = img
	s.results = results
	capturePath, _ := config.LastCapturePath()
	if capturePath != "" {
		_ = capture.SavePNG(capturePath, img)
	}

	s.renderResultCards(results)
	s.setStatus(fmt.Sprintf("%s完成，识别到 %d 组", source, len(results)))
	return nil
}

func (s *uiState) renderResultCards(results []analyzer.Result) {
	objects := make([]fyne.CanvasObject, 0, len(results))
	for idx, result := range results {
		details := make([]fyne.CanvasObject, 0, len(result.Candidates)+1)
		metrics := container.NewGridWithColumns(2,
			buildMetricRow("Size", fmt.Sprintf("%.3f", result.Measurement.Size)),
			buildMetricRow("Weight", fmt.Sprintf("%.3f", result.Measurement.Weight)),
		)
		details = append(details, metrics)

		for candidateIndex, candidate := range result.Candidates {
			title := fmt.Sprintf("%d. %s (%s)", candidateIndex+1, candidate.Pet, candidate.PetID)
			meta := fmt.Sprintf("概率 %.2f%% | 参考尺寸 %s | 参考重量 %s | %s", candidate.Probability, candidate.EggDiameter, candidate.EggWeight, candidate.MatchType)
			line := widget.NewLabel(title + "\n" + meta)
			line.Wrapping = fyne.TextWrapWord
			if candidateIndex == 0 {
				line.Importance = widget.HighImportance
			}
			details = append(details, s.buildCandidateRow(candidate.PetID, line))
		}

		objects = append(objects, buildSurfaceCard(
			container.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("蛋 %d", idx+1), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				container.NewVBox(details...),
			),
			color.NRGBA{R: 0xFB, G: 0xF8, B: 0xF1, A: 0x94},
			color.NRGBA{R: 0xE0, G: 0xD1, B: 0xB6, A: 0xA4},
			20,
		))
	}
	if len(objects) == 0 {
		objects = append(objects, s.buildEmptyResultCard())
	}
	s.resultList.Objects = objects
	s.resultList.Refresh()
}

func (s *uiState) buildEmptyResultCard() fyne.CanvasObject {
	label := widget.NewLabel("暂无分析结果。\n先截图或导入图片，结果会按蛋分组显示。")
	label.Wrapping = fyne.TextWrapWord
	return buildSurfaceCard(label, color.NRGBA{R: 0xFB, G: 0xF8, B: 0xF1, A: 0x94}, color.NRGBA{R: 0xE0, G: 0xD1, B: 0xB6, A: 0xA4}, 20)
}

func (s *uiState) buildCandidateRow(petID string, content fyne.CanvasObject) fyne.CanvasObject {
	sprite := canvas.NewImageFromImage(buildPetPlaceholderImage())
	sprite.FillMode = canvas.ImageFillContain
	sprite.SetMinSize(fyne.NewSize(72, 72))
	s.loadCandidateImage(petID, sprite)

	return container.NewBorder(nil, nil, container.NewPadded(sprite), nil, content)
}

func (s *uiState) loadCandidateImage(petID string, target *canvas.Image) {
	if s.atlas == nil || petID == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		img, err := s.atlas.Load(ctx, petID)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Printf("load pet image %s failed: %v\n", petID, err)
			}
			return
		}

		target.Image = img
		target.Refresh()
	}()
}

func buildMetricRow(name, value string) fyne.CanvasObject {
	label := widget.NewLabel(name)
	label.Importance = widget.MediumImportance
	number := widget.NewLabelWithStyle(value, fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	return container.NewPadded(container.NewBorder(nil, nil, label, nil, number))
}

func (s *uiState) warmAtlasIndex() {
	if s.atlas == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.atlas.WarmIndex(ctx); err != nil {
			fmt.Printf("warm atlas index failed: %v\n", err)
		}
	}()
}

func (s *uiState) run(name string, fn func() error) {
	go func() {
		if err := fn(); err != nil {
			s.setStatus(name + "失败")
			s.showError(err)
			return
		}
		s.setStatus(name + "完成")
	}()
}

func (s *uiState) setStatus(message string) {
	s.panelStatus.SetText(message)
}

func (s *uiState) showError(err error) {
	fmt.Printf("error: %v\n", err)
	dialog.ShowError(err, s.dialogParent())
}

func (s *uiState) dialogParent() fyne.Window {
	if s.panelVisible {
		return s.panelWindow
	}
	if s.resultVisible {
		return s.resultWindow
	}
	return s.bubbleWindow
}

func (s *uiState) togglePanelWindow() {
	if s.panelVisible {
		s.hidePanelWindow()
		return
	}
	s.showPanelWindow()
}

func (s *uiState) showBubbleWindow() {
	if !s.bubblePrimed {
		s.bubblePrimed = true
		s.bubbleWindow.Show()
		configureNativeWindow(s.bubbleWindow, nativeWindowStyle{
			CornerRadius:        30,
			Floating:            true,
			Transparent:         true,
			MovableByBackground: true,
		})
		s.bubbleWindow.Hide()
		go func() {
			time.Sleep(18 * time.Millisecond)
			s.showBubbleWindow()
		}()
		return
	}
	s.bubbleWindow.Show()
	style := nativeWindowStyle{
		CornerRadius:        30,
		Floating:            true,
		Transparent:         true,
		MovableByBackground: true,
	}
	configureNativeWindow(s.bubbleWindow, style)
	s.restoreBubblePosition()
	s.bubbleWindow.RequestFocus()
	go func() {
		for _, delay := range []time.Duration{18 * time.Millisecond, 72 * time.Millisecond} {
			time.Sleep(delay)
			configureNativeWindow(s.bubbleWindow, style)
			s.restoreBubblePosition()
		}
	}()
}

func (s *uiState) hideBubbleWindow() {
	s.bubbleWindow.Hide()
}

func (s *uiState) showPanelWindow() {
	s.rememberBubbleAnchor()
	s.hideBubbleWindow()
	s.panelWindow.Show()
	style := nativeWindowStyle{
		CornerRadius:        34,
		Floating:            true,
		Transparent:         true,
		MovableByBackground: true,
	}
	configureNativeWindow(s.panelWindow, style)
	s.placeWindowNearAnchor(s.panelWindow, 14)
	s.panelVisible = true
	s.ensureWindowReady(s.panelWindow, style, 14)
}

func (s *uiState) hidePanelWindow() {
	s.panelWindow.Hide()
	s.panelVisible = false
	s.showBubbleWindow()
}

func (s *uiState) showResultWindow() {
	if !s.panelVisible {
		s.rememberBubbleAnchor()
	}
	if s.panelVisible {
		s.panelWindow.Hide()
		s.panelVisible = false
	}
	s.hideBubbleWindow()
	s.resultWindow.Show()
	style := nativeWindowStyle{
		CornerRadius:        36,
		Floating:            true,
		Transparent:         true,
		MovableByBackground: true,
	}
	configureNativeWindow(s.resultWindow, style)
	s.placeWindowNearAnchor(s.resultWindow, 18)
	s.resultVisible = true
	s.ensureWindowReady(s.resultWindow, style, 18)
}

func (s *uiState) hideResultWindow() {
	s.resultWindow.Hide()
	s.resultVisible = false
	if !s.panelVisible {
		s.showBubbleWindow()
	}
}

func (s *uiState) restoreBubblePosition() {
	if s.bubbleRestored {
		return
	}
	s.bubbleRestored = true

	cfg := s.config.Get()
	if cfg.BubblePositionSet {
		setNativeWindowOrigin(s.bubbleWindow, float32(cfg.BubbleX), float32(cfg.BubbleY))
	}
	s.rememberBubbleAnchor()
}

func (s *uiState) rememberBubbleAnchor() {
	frame, ok := getNativeWindowFrame(s.bubbleWindow)
	if !ok {
		return
	}
	s.bubbleAnchor = frame
	s.bubbleAnchorSet = true

	cfg := s.config.Get()
	x := int(frame.X)
	y := int(frame.Y)
	if cfg.BubblePositionSet && cfg.BubbleX == x && cfg.BubbleY == y {
		return
	}
	cfg.BubblePositionSet = true
	cfg.BubbleX = x
	cfg.BubbleY = y
	if err := s.config.Save(cfg); err != nil {
		fmt.Printf("save bubble position failed: %v\n", err)
	}
}

func (s *uiState) rememberBubbleAnchorSoon() {
	go func() {
		time.Sleep(18 * time.Millisecond)
		s.rememberBubbleAnchor()
	}()
}

func (s *uiState) placeWindowNearAnchor(win fyne.Window, gap float32) {
	if !s.bubbleAnchorSet {
		return
	}

	frame, ok := getNativeWindowFrame(win)
	if !ok {
		return
	}
	visible, ok := getNativeVisibleFrame(win)
	if !ok {
		return
	}

	x := s.bubbleAnchor.X + (s.bubbleAnchor.Width-frame.Width)/2
	minX := visible.X + 12
	maxX := visible.X + visible.Width - frame.Width - 12
	if maxX < minX {
		maxX = minX
	}
	if x < minX {
		x = minX
	}
	if x > maxX {
		x = maxX
	}

	y := s.bubbleAnchor.Y - frame.Height - gap
	if y < visible.Y+12 {
		y = s.bubbleAnchor.Y + s.bubbleAnchor.Height + gap
	}
	maxY := visible.Y + visible.Height - frame.Height - 12
	if maxY < visible.Y+12 {
		maxY = visible.Y + 12
	}
	if y > maxY {
		y = maxY
	}
	if y < visible.Y+12 {
		y = visible.Y + 12
	}

	setNativeWindowOrigin(win, x, y)
}

func (s *uiState) ensureWindowReady(win fyne.Window, style nativeWindowStyle, gap float32) {
	win.RequestFocus()
	go func() {
		for _, delay := range []time.Duration{24 * time.Millisecond, 90 * time.Millisecond} {
			time.Sleep(delay)
			configureNativeWindow(win, style)
			if gap >= 0 {
				s.placeWindowNearAnchor(win, gap)
			}
			win.RequestFocus()
		}
	}()
}

func (s *uiState) quit() {
	if s.quitting {
		return
	}
	s.quitting = true
	s.cleanup()
	s.app.Quit()
}

func (s *uiState) cleanup() {
	s.cleanupOnce.Do(func() {
		if closer, ok := s.ocr.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	})
}

func buildPetPlaceholderImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 180, 180))
	for y := 0; y < 180; y++ {
		for x := 0; x < 180; x++ {
			img.Set(x, y, color.RGBA{R: 239, G: 232, B: 219, A: 255})
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
		return color.NRGBA{R: 0xF5, G: 0xF7, B: 0xFB, A: 0xFF}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0xFB, G: 0xFC, B: 0xFE, A: 0xFF}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0xD9, G: 0xE0, B: 0xEA, A: 0xFF}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0xF7, G: 0xF9, B: 0xFD, A: 0xFF}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0x17, G: 0x1D, B: 0x29, A: 0xFF}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x7E, G: 0x89, B: 0x99, A: 0xFF}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x6F, G: 0x7B, B: 0x8C, A: 0xFF}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0xC2, G: 0xCB, B: 0xD8, A: 0xCC}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x53, G: 0x86, B: 0xFF, A: 0xFF}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x53, G: 0x86, B: 0xFF, A: 0x18}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0x53, G: 0x86, B: 0xFF, A: 0x2E}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0xDB, G: 0xE3, B: 0xEE, A: 0xFF}
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
