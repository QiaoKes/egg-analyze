# 洛克王国蛋分析工具

Windows 桌面工具，使用 Go 开发。

## 已实现

- 全局快捷键截图，快捷键可配置
- 从 `https://rocom.mfsky.qzz.io/data/egg-measurements-final.json` 拉取最新蛋尺寸数据并缓存
- 复刻参考网站的本地概率计算逻辑
- 调用 Windows 内置 OCR 识别截图中的蛋尺寸和重量
- 图形界面展示分析结果
- 关闭窗口隐藏到 Windows 托盘
- 支持打开本地图片直接分析

## 运行

```powershell
go run .
```

或构建：

```powershell
go build .
```

## 使用说明

1. 启动程序
2. 点击“刷新网站数据”确认数据源可用
3. 点击“打开图片分析”测试 `picture` 中的样例图
4. 点击“快捷键截图分析”或使用你配置的全局快捷键
5. 在右侧查看识别到的尺寸/重量和候选精灵概率

默认快捷键：

```text
Ctrl+Shift+Q
```

## 当前实现说明

- 数据源使用站点公开 JSON，不直接依赖 GitHub 仓库内的静态快照
- 首版默认截取主显示器整屏
- OCR 依赖 Windows 内置能力，不需要额外安装 Tesseract
- 当前识别策略是“先找蛋标题，再在附近提取数字”，比整图全文 OCR 更稳

## 当前限制

- 多显示器区域框选还未实现
- 多蛋截图里如果单个卡片分辨率过低，可能漏识别部分蛋
- 托盘图标是代码生成的简易图标，后续可以替换成正式 `.ico`

## 关键文件

- `main.go`: Windows GUI、托盘、交互入口
- `internal/analyzer`: 蛋标题锚点、OCR 数字提取、候选分析
- `internal/ocr`: Windows OCR 调用
- `internal/rocom`: 站点数据同步与概率算法
- `docs/progress.md`: 实施计划与卡点记录
