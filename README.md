# 洛克王国蛋分析工具

跨平台桌面工具，使用 Go 开发。

## 已实现

- 全局快捷键截图，快捷键可配置
- 从 `https://rocom.mfsky.qzz.io/data/egg-measurements-final.json` 拉取最新蛋尺寸数据并缓存
- 复刻参考网站的本地概率计算逻辑
- 调用 RapidOCR 识别截图中的蛋尺寸和重量
- 调用 `flameshot` 完成截图框选
- 图形界面展示分析结果
- 关闭窗口隐藏到系统托盘
- 支持打开本地图片直接分析

## 运行

源码直接运行时，先安装 OCR 依赖和 `flameshot`：

```bash
python3 -m venv .venv
./.venv/bin/python3 -m pip install -r requirements-ocr.txt
```

然后运行：

```bash
go run .
```

或构建：

```bash
go build .
```

## 使用说明

1. 启动程序
2. 点击“刷新网站数据”确认数据源可用
3. 点击“打开图片”测试 `picture` 中的样例图
4. 点击“截图分析”或使用你配置的全局快捷键
5. 进入 `flameshot` 框选界面，选中要识别的区域
6. 在右侧查看识别到的尺寸/重量和候选精灵概率

默认快捷键：

```text
Ctrl+Shift+Q
```

## 当前实现说明

- 数据源使用站点公开 JSON，不直接依赖 GitHub 仓库内的静态快照
- 截图统一交给 `flameshot gui -p <file> -s`
- OCR 使用 Python 常驻 `RapidOCR`
- 当前识别策略以数字提取为中心，不依赖固定蛋标题
- 程序会优先使用随包分发的 `flameshot`，其次才是系统 `PATH` 中的版本
- 程序会优先使用随包分发的 `python/` 运行时，随后才是 `EGG_ANALYZE_PYTHON` 和本地环境

## 当前限制

- 源码直接运行时需要本机已安装 `flameshot`，或手动设置 `EGG_ANALYZE_FLAMESHOT_BIN`
- 源码直接运行时需要本机 Python 环境，或手动设置 `EGG_ANALYZE_PYTHON`
- 非 Windows 平台下，应用内“全局热键”后端仍未实现系统级注册
- 多蛋截图里如果单个卡片分辨率过低，可能漏识别部分蛋
- GitHub Actions 打包产物会同时内置 `flameshot` 和 Python OCR 运行时，体积会明显变大

## 自动打包

- GitHub Actions 工作流在 [package.yml](.github/workflows/package.yml)
- 会在 `develop` 分支 push 和 `v*` tag 上自动构建
- 当前会产出：
  - `windows-amd64`
  - `macos-amd64`
  - `macos-arm64`
- 每个包里都会带对应平台的官方 `flameshot` 二进制
- 每个包里都会带对应平台的 Python 3.11 运行时和 `rapidocr-onnxruntime`

## 关键文件

- `main.go`: GUI、托盘、交互入口
- `internal/analyzer`: OCR 数字提取、候选分析
- `internal/ocr`: RapidOCR 调用
- `internal/capture`: Flameshot 截图调用
- `internal/rocom`: 站点数据同步与概率算法
- `docs/progress.md`: 实施计划与卡点记录
