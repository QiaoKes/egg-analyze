package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"strings"
	"time"

	"egg-analyze/internal/analyzer"
	"egg-analyze/internal/capture"
	"egg-analyze/internal/config"
	"egg-analyze/internal/hotkey"
	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type uiState struct {
	window       *walk.MainWindow
	notifyIcon   *walk.NotifyIcon
	statusLabel  *walk.Label
	sourceLabel  *walk.Label
	hotkeyEdit   *walk.LineEdit
	logEdit      *walk.TextEdit
	resultsEdit  *walk.TextEdit
	previewView  *walk.ImageView
	previewImage *walk.Bitmap

	config     *config.Manager
	hotkeys    *hotkey.Manager
	ocr        *ocr.Recognizer
	engine     *rocom.Engine
	analyzeSvc *analyzer.Service

	currentImage image.Image
	quitting     bool
}

func main() {
	cfgMgr, err := config.Load()
	if err != nil {
		panic(err)
	}

	state := &uiState{
		config:  cfgMgr,
		hotkeys: hotkey.NewManager(),
		ocr:     ocr.New(),
	}

	if err := state.buildUI(); err != nil {
		panic(err)
	}
	if err := state.setupTray(); err != nil {
		state.log(fmt.Sprintf("托盘初始化失败: %v", err))
	}
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

	state.window.Run()
	state.hotkeys.Close()
	if state.notifyIcon != nil {
		state.notifyIcon.Dispose()
	}
	if state.previewImage != nil {
		state.previewImage.Dispose()
	}
}

func (s *uiState) buildUI() error {
	cfg := s.config.Get()

	return MainWindow{
		AssignTo: &s.window,
		Title:    "洛克王国蛋分析",
		MinSize:  Size{1180, 760},
		Layout:   VBox{},
		Children: []Widget{
			HSplitter{
				Children: []Widget{
					Composite{
						Layout: VBox{},
						Children: []Widget{
							GroupBox{
								Title:  "状态",
								Layout: VBox{},
								Children: []Widget{
									Label{AssignTo: &s.statusLabel, Text: "准备就绪"},
									Label{AssignTo: &s.sourceLabel, Text: "数据源：未加载"},
								},
							},
							GroupBox{
								Title:  "操作",
								Layout: VBox{},
								Children: []Widget{
									PushButton{
										Text: "快捷键截图分析",
										OnClicked: func() {
											s.run("截图分析", s.captureAndAnalyze)
										},
									},
									PushButton{
										Text:      "打开图片分析",
										OnClicked: s.openImageDialog,
									},
									PushButton{
										Text: "刷新网站数据",
										OnClicked: func() {
											s.run("刷新数据", s.reloadDataset)
										},
									},
									PushButton{
										Text:      "保存当前截图",
										OnClicked: s.saveCurrentCapture,
									},
								},
							},
							GroupBox{
								Title:  "快捷键",
								Layout: VBox{},
								Children: []Widget{
									Label{Text: "全局截图热键"},
									LineEdit{AssignTo: &s.hotkeyEdit, Text: cfg.Hotkey},
									PushButton{
										Text:      "保存快捷键",
										OnClicked: s.saveHotkey,
									},
									Label{Text: "关闭窗口会隐藏到托盘。"},
								},
							},
							GroupBox{
								Title:  "截图预览",
								Layout: VBox{},
								Children: []Widget{
									ImageView{
										AssignTo: &s.previewView,
										Mode:     ImageViewModeShrink,
										MinSize:  Size{520, 320},
									},
								},
							},
							GroupBox{
								Title:  "日志",
								Layout: VBox{},
								Children: []Widget{
									TextEdit{
										AssignTo: &s.logEdit,
										ReadOnly: true,
										VScroll:  true,
										MinSize:  Size{520, 180},
										Text:     "启动完成，等待操作。",
									},
								},
							},
						},
					},
					Composite{
						Layout: VBox{},
						Children: []Widget{
							GroupBox{
								Title:  "分析结果",
								Layout: VBox{},
								Children: []Widget{
									TextEdit{
										AssignTo: &s.resultsEdit,
										ReadOnly: true,
										VScroll:  true,
										MinSize:  Size{520, 650},
										Text:     "暂无分析结果",
									},
								},
							},
						},
					},
				},
			},
		},
	}.Create()
}

func (s *uiState) setupTray() error {
	icon, err := walk.NewIconFromImage(buildTrayImage())
	if err != nil {
		return err
	}

	notifyIcon, err := walk.NewNotifyIcon(s.window)
	if err != nil {
		return err
	}

	if err := notifyIcon.SetIcon(icon); err != nil {
		return err
	}
	if err := notifyIcon.SetToolTip("洛克王国蛋分析"); err != nil {
		return err
	}

	notifyIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			s.showWindow()
		}
	})

	showAction := walk.NewAction()
	showAction.SetText("显示窗口")
	showAction.Triggered().Attach(s.showWindow)

	captureAction := walk.NewAction()
	captureAction.SetText("截图分析")
	captureAction.Triggered().Attach(func() {
		s.run("截图分析", s.captureAndAnalyze)
	})

	refreshAction := walk.NewAction()
	refreshAction.SetText("刷新数据")
	refreshAction.Triggered().Attach(func() {
		s.run("刷新数据", s.reloadDataset)
	})

	exitAction := walk.NewAction()
	exitAction.SetText("退出")
	exitAction.Triggered().Attach(func() {
		s.quitting = true
		s.hotkeys.Close()
		notifyIcon.Dispose()
		s.window.Close()
	})

	menu := notifyIcon.ContextMenu().Actions()
	menu.Add(showAction)
	menu.Add(captureAction)
	menu.Add(refreshAction)
	menu.Add(exitAction)

	if err := notifyIcon.SetVisible(true); err != nil {
		return err
	}

	s.notifyIcon = notifyIcon
	return nil
}

func (s *uiState) attachCloseBehavior() {
	s.window.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if s.quitting {
			return
		}
		*canceled = true
		s.window.Hide()
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

	s.runOnUI(func() {
		s.sourceLabel.SetText(fmt.Sprintf("数据源：%s | 记录 %d | 更新时间 %s", source, info.RecordCount, info.UpdatedAt.Format("2006-01-02 15:04:05")))
	})
	s.log(fmt.Sprintf("数据已加载，来源=%s，记录=%d", info.URL, info.RecordCount))
	return nil
}

func (s *uiState) captureAndAnalyze() error {
	if s.analyzeSvc == nil {
		return fmt.Errorf("数据尚未加载")
	}
	img, err := capture.CapturePrimary()
	if err != nil {
		return err
	}
	return s.handleImage(img, "屏幕截图")
}

func (s *uiState) handleImage(img image.Image, source string) error {
	results, rawLines, err := s.analyzeSvc.AnalyzeImage(context.Background(), img)
	if err != nil {
		return fmt.Errorf("%w；OCR原始行数=%d", err, len(rawLines))
	}

	s.currentImage = img
	capturePath, _ := config.LastCapturePath()
	if capturePath != "" {
		_ = capture.SavePNG(capturePath, img)
	}

	s.runOnUI(func() {
		s.updatePreview(img)
		s.renderResults(results, source, len(rawLines))
	})
	s.log(fmt.Sprintf("%s 完成，识别到 %d 组尺寸/重量", source, len(results)))
	return nil
}

func (s *uiState) renderResults(results []analyzer.Result, source string, rawLineCount int) {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s | OCR 行数 %d | 命中 %d 组数据\n\n", source, rawLineCount, len(results)))
	for idx, result := range results {
		builder.WriteString(fmt.Sprintf("蛋 %d\n", idx+1))
		builder.WriteString(fmt.Sprintf("尺寸: %.3f\n", result.Measurement.Size))
		builder.WriteString(fmt.Sprintf("重量: %.3f\n", result.Measurement.Weight))
		builder.WriteString("候选:\n")
		for _, candidate := range result.Candidates {
			builder.WriteString(fmt.Sprintf("  - %s (%s) %.2f%% | 参考尺寸 %s | 参考重量 %s | %s\n", candidate.Pet, candidate.PetID, candidate.Probability, candidate.EggDiameter, candidate.EggWeight, candidate.MatchType))
		}
		builder.WriteString("\n")
	}
	s.resultsEdit.SetText(builder.String())
}

func (s *uiState) updatePreview(img image.Image) {
	if s.previewImage != nil {
		s.previewImage.Dispose()
		s.previewImage = nil
	}
	bitmap, err := walk.NewBitmapFromImage(img)
	if err != nil {
		s.log(fmt.Sprintf("预览图更新失败: %v", err))
		return
	}
	s.previewImage = bitmap
	_ = s.previewView.SetImage(bitmap)
}

func (s *uiState) openImageDialog() {
	dialog := new(walk.FileDialog)
	dialog.Filter = "图片文件 (*.png;*.jpg;*.jpeg)|*.png;*.jpg;*.jpeg"

	ok, err := dialog.ShowOpen(s.window)
	if err != nil {
		s.log(fmt.Sprintf("打开图片对话框失败: %v", err))
		return
	}
	if !ok {
		return
	}

	file, err := os.Open(dialog.FilePath)
	if err != nil {
		s.log(fmt.Sprintf("读取图片失败: %v", err))
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		s.log(fmt.Sprintf("解析图片失败: %v", err))
		return
	}

	s.run("图片分析", func() error {
		return s.handleImage(img, dialog.FilePath)
	})
}

func (s *uiState) saveCurrentCapture() {
	if s.currentImage == nil {
		s.log("当前没有可保存的截图")
		return
	}

	dialog := new(walk.FileDialog)
	dialog.Filter = "PNG 文件 (*.png)|*.png"
	dialog.FilePath = "egg-capture.png"

	ok, err := dialog.ShowSave(s.window)
	if err != nil {
		s.log(fmt.Sprintf("保存对话框失败: %v", err))
		return
	}
	if !ok {
		return
	}

	file, err := os.Create(dialog.FilePath)
	if err != nil {
		s.log(fmt.Sprintf("创建文件失败: %v", err))
		return
	}
	defer file.Close()

	if err := png.Encode(file, s.currentImage); err != nil {
		s.log(fmt.Sprintf("保存截图失败: %v", err))
		return
	}
	s.log(fmt.Sprintf("截图已保存到 %s", dialog.FilePath))
}

func (s *uiState) saveHotkey() {
	newCfg := s.config.Get()
	newCfg.Hotkey = strings.TrimSpace(s.hotkeyEdit.Text())
	if err := s.config.Save(newCfg); err != nil {
		s.log(fmt.Sprintf("保存快捷键失败: %v", err))
		return
	}
	if err := s.registerHotkey(newCfg.Hotkey); err != nil {
		s.log(fmt.Sprintf("重新注册快捷键失败: %v", err))
		return
	}
	s.log("快捷键已更新")
}

func (s *uiState) registerHotkey(hotkeyText string) error {
	return s.hotkeys.Register(hotkeyText, func() {
		if s.notifyIcon != nil {
			_ = s.notifyIcon.ShowInfo("洛克王国蛋分析", "收到截图热键，开始分析主屏截图。")
		}
		s.run("热键截图分析", s.captureAndAnalyze)
	})
}

func (s *uiState) run(name string, fn func() error) {
	s.setStatus(name + "处理中")
	go func() {
		err := fn()
		if err != nil {
			s.setStatus(name + "失败")
			s.log(fmt.Sprintf("%s失败: %v", name, err))
			return
		}
		s.setStatus(name + "完成")
	}()
}

func (s *uiState) runOnUI(fn func()) {
	if s.window == nil {
		fn()
		return
	}
	s.window.Synchronize(fn)
}

func (s *uiState) log(message string) {
	s.runOnUI(func() {
		current := strings.TrimSpace(s.logEdit.Text())
		timestamped := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message)
		if current == "" {
			s.logEdit.SetText(timestamped)
			return
		}
		s.logEdit.SetText(current + "\r\n" + timestamped)
	})
}

func (s *uiState) setStatus(message string) {
	s.runOnUI(func() {
		s.statusLabel.SetText(message)
	})
}

func (s *uiState) showWindow() {
	s.runOnUI(func() {
		s.window.Show()
		s.window.SetFocus()
	})
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
