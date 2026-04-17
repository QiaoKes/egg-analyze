import 'dart:io';

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:window_manager/window_manager.dart';

class DesktopWindowController with WindowListener {
  static const _bubbleXKey = 'desktop_bubble_x';
  static const _bubbleYKey = 'desktop_bubble_y';

  bool _initialized = false;
  bool _bubbleMode = false;
  String _lastFullRoute = '/';
  Rect? _lastFullBounds;

  bool get isDesktop => Platform.isMacOS || Platform.isWindows;
  bool get isBubbleMode => _bubbleMode;
  String get restoreRoute => _lastFullRoute;

  Future<void> initialize() async {
    if (!isDesktop || _initialized) {
      return;
    }
    _initialized = true;
    await windowManager.ensureInitialized();
    windowManager.addListener(this);
    const options = WindowOptions(
      size: Size(68, 68),
      minimumSize: Size(68, 68),
      maximumSize: Size(68, 68),
      center: false,
      backgroundColor: Colors.transparent,
      titleBarStyle: TitleBarStyle.hidden,
      windowButtonVisibility: false,
      alwaysOnTop: true,
      skipTaskbar: true,
    );
    await windowManager.waitUntilReadyToShow(options, () async {
      await windowManager.setAsFrameless();
      await _restoreBubblePositionOrDefault();
      await windowManager.show();
      await windowManager.focus();
    });
    _bubbleMode = true;
  }

  void registerRoute(String route) {
    if (route == '/bubble') {
      return;
    }
    _lastFullRoute = route;
  }

  Future<void> enterBubbleMode() async {
    if (!isDesktop || _bubbleMode) {
      return;
    }
    _lastFullBounds ??= await windowManager.getBounds();
    _bubbleMode = true;
    await windowManager.setAsFrameless();
    await windowManager.setAlwaysOnTop(true);
    await windowManager.setSkipTaskbar(true);
    await windowManager.setResizable(false);
    await windowManager.setMinimizable(false);
    await windowManager.setMaximizable(false);
    await windowManager.setBackgroundColor(Colors.transparent);
    await windowManager.setMinimumSize(const Size(68, 68));
    await windowManager.setMaximumSize(const Size(68, 68));
    await windowManager.setSize(const Size(68, 68));
    await _restoreBubblePositionOrDefault();
    await windowManager.focus();
  }

  Future<void> exitBubbleMode() async {
    if (!isDesktop || !_bubbleMode) {
      return;
    }
    _bubbleMode = false;
    await windowManager.setAlwaysOnTop(false);
    await windowManager.setSkipTaskbar(false);
    await windowManager.setResizable(true);
    await windowManager.setMinimizable(true);
    await windowManager.setMaximizable(true);
    await windowManager.setTitleBarStyle(
      TitleBarStyle.normal,
      windowButtonVisibility: true,
    );
    await windowManager.setBackgroundColor(Colors.white);
    await windowManager.setMinimumSize(const Size(900, 640));
    await windowManager.setMaximumSize(const Size(-1, -1));
    if (_lastFullBounds != null) {
      await windowManager.setBounds(_lastFullBounds!, animate: true);
    } else {
      await windowManager.setSize(const Size(1040, 720));
      await windowManager.center();
    }
    await windowManager.focus();
  }

  @override
  void onWindowMoved() {
    if (!_bubbleMode) {
      return;
    }
    _persistBubblePosition();
  }

  Future<void> disposeController() async {
    if (!_initialized) {
      return;
    }
    windowManager.removeListener(this);
  }

  Future<void> _restoreBubblePositionOrDefault() async {
    final prefs = await SharedPreferences.getInstance();
    final dx = prefs.getDouble(_bubbleXKey);
    final dy = prefs.getDouble(_bubbleYKey);
    if (dx != null && dy != null) {
      await windowManager.setPosition(Offset(dx, dy));
      return;
    }
    await windowManager.setAlignment(Alignment.topRight, animate: true);
  }

  Future<void> _persistBubblePosition() async {
    final position = await windowManager.getPosition();
    final prefs = await SharedPreferences.getInstance();
    await prefs.setDouble(_bubbleXKey, position.dx);
    await prefs.setDouble(_bubbleYKey, position.dy);
  }
}
