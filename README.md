# egg-analyze-v2

Flutter 纯栈版 v2 骨架，目标平台为 `macOS / Windows / Android`。

## 结构

```text
egg-analyze-v2/
  app/
  packages/
    egg_core/
    egg_ocr/
    egg_data/
```

## 本地启动

当前目录已经补齐 `android / macos / windows` 宿主工程。

首次拉取后，在 `app/` 目录执行：

```bash
flutter create . --platforms=android,macos,windows
flutter pub get
```

如果需要同步本地包依赖，再分别执行：

```bash
cd packages/egg_core && flutter pub get
cd ../egg_ocr && flutter pub get
cd ../egg_data && flutter pub get
```

## 实现范围

- `egg_core`：匹配引擎、OCR 数值提取、概率计算
- `egg_ocr`：桌面 `platform_ocr` + Android `google_mlkit_text_recognition`
- `egg_data`：`Pets.json`、头像资源缓存
- `app`：首页、结果页、设置页三页骨架

## 当前限制

- 未接入桌面截图、悬浮球、全局热键
- Android 分享入口只补了 Manifest 与 Dart 监听，仍需真机联调
- 还没有接入现有 `picture/` 真实样本图的自动回归夹具
