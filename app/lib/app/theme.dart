import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

const _appFontFamily = 'Noto Sans CJK SC';

ThemeData buildAppTheme() {
  const seed = Color(0xFF2762FF);
  final typography = Typography.material2021(platform: defaultTargetPlatform);
  final fontFallback = _systemFontFallback(defaultTargetPlatform);
  return ThemeData(
    colorScheme: ColorScheme.fromSeed(seedColor: seed),
    useMaterial3: true,
    fontFamily: _appFontFamily,
    scaffoldBackgroundColor: const Color(0xFFF6F8FC),
    textTheme: typography.black.apply(
      fontFamily: _appFontFamily,
      fontFamilyFallback: fontFallback,
    ),
    primaryTextTheme: typography.white.apply(
      fontFamily: _appFontFamily,
      fontFamilyFallback: fontFallback,
    ),
    cardTheme: const CardThemeData(
      elevation: 0,
      margin: EdgeInsets.zero,
    ),
  );
}

List<String> _systemFontFallback(TargetPlatform platform) {
  switch (platform) {
    case TargetPlatform.windows:
      return const [
        'Noto Sans CJK SC',
        'Noto Sans SC',
        'Segoe UI',
        'Microsoft YaHei UI',
        'Microsoft YaHei',
      ];
    case TargetPlatform.macOS:
    case TargetPlatform.iOS:
      return const [
        'Noto Sans CJK SC',
        'Noto Sans SC',
        '.SF NS Text',
        'PingFang SC',
        'Helvetica Neue',
      ];
    case TargetPlatform.android:
      return const [
        'Noto Sans CJK SC',
        'Noto Sans SC',
        'sans-serif',
        'Noto Sans',
      ];
    case TargetPlatform.linux:
      return const [
        'Noto Sans CJK SC',
        'Noto Sans SC',
        'Noto Sans',
        'Ubuntu',
      ];
    case TargetPlatform.fuchsia:
      return const ['Noto Sans CJK SC', 'Noto Sans', 'sans-serif'];
  }
}
