import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:screen_retriever/screen_retriever.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:window_manager/window_manager.dart';

class DesktopWindowController with WindowListener {
  static const _bubbleXKey = 'desktop_bubble_x';
  static const _bubbleYKey = 'desktop_bubble_y';
  static const Size _bubbleSize = Size(60, 60);
  static const Size _defaultFullSize = Size(920, 640);
  static const Size _minimumFullSize = Size(760, 560);
  static const Duration _transitionDuration = Duration(milliseconds: 120);

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
      size: _bubbleSize,
      minimumSize: _bubbleSize,
      maximumSize: _bubbleSize,
      center: false,
      backgroundColor: Colors.transparent,
      titleBarStyle: TitleBarStyle.hidden,
      windowButtonVisibility: false,
      alwaysOnTop: true,
      skipTaskbar: true,
    );
    await windowManager.waitUntilReadyToShow(options, () async {
      await windowManager.setAsFrameless();
      await windowManager.setHasShadow(false);
      await windowManager.setAlwaysOnTop(true);
      await windowManager.setResizable(false);
      await windowManager.setMinimizable(false);
      await windowManager.setMaximizable(false);
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
    _lastFullBounds = await windowManager.getBounds();
    _bubbleMode = true;
    await windowManager.setAsFrameless();
    await windowManager.setHasShadow(false);
    await windowManager.setAlwaysOnTop(true);
    await windowManager.setSkipTaskbar(true);
    await windowManager.setResizable(false);
    await windowManager.setMinimizable(false);
    await windowManager.setMaximizable(false);
    await windowManager.setBackgroundColor(Colors.transparent);
    await windowManager.setMinimumSize(_bubbleSize);
    await windowManager.setMaximumSize(_bubbleSize);
    await windowManager.setSize(_bubbleSize);
    await _restoreBubblePositionOrDefault();
    await windowManager.focus();
  }

  Future<void> exitBubbleMode() async {
    if (!isDesktop || !_bubbleMode) {
      return;
    }
    _bubbleMode = false;
    await windowManager.setHasShadow(true);
    await windowManager.setAlwaysOnTop(true);
    await windowManager.setSkipTaskbar(false);
    await windowManager.setResizable(true);
    await windowManager.setMinimizable(true);
    await windowManager.setMaximizable(true);
    await windowManager.setTitleBarStyle(
      TitleBarStyle.normal,
      windowButtonVisibility: true,
    );
    await windowManager.setBackgroundColor(Colors.white);
    await windowManager.setMinimumSize(_minimumFullSize);
    await windowManager.setMaximumSize(const Size(-1, -1));
    final targetSize = _lastFullBounds?.size ?? _defaultFullSize;
    final targetBounds = _lastFullBounds != null
        ? _clampRectToVisibleArea(
            await _currentDisplayVisibleRect(_lastFullBounds!),
            _lastFullBounds!,
          )
        : await _deriveBoundsNearBubble(targetSize);
    await windowManager.setBounds(targetBounds);
    await windowManager.focus();
  }

  Future<void> transitionToBubble(FutureOr<void> Function() routeChange) async {
    if (!isDesktop) {
      await routeChange();
      return;
    }
    await _animateOpacity(from: 1, to: 0);
    await routeChange();
    await Future<void>.delayed(const Duration(milliseconds: 16));
    await enterBubbleMode();
    await _animateOpacity(from: 0, to: 1);
  }

  Future<void> transitionToFull(FutureOr<void> Function() routeChange) async {
    if (!isDesktop) {
      await routeChange();
      return;
    }
    await _animateOpacity(from: 1, to: 0);
    await exitBubbleMode();
    await routeChange();
    await Future<void>.delayed(const Duration(milliseconds: 16));
    await _animateOpacity(from: 0, to: 1);
  }

  @override
  void onWindowMoved() {
    if (!_bubbleMode) {
      return;
    }
    unawaited(_persistBubblePosition());
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
    await windowManager.setAlignment(Alignment.topRight);
  }

  Future<void> _persistBubblePosition() async {
    final position = await windowManager.getPosition();
    final prefs = await SharedPreferences.getInstance();
    await prefs.setDouble(_bubbleXKey, position.dx);
    await prefs.setDouble(_bubbleYKey, position.dy);
  }

  Future<Rect> _deriveBoundsNearBubble(Size targetSize) async {
    final bubbleBounds = await windowManager.getBounds();
    final displays = await screenRetriever.getAllDisplays();
    final primaryDisplay = await screenRetriever.getPrimaryDisplay();
    final bubbleCenter = Offset(
      bubbleBounds.left + (bubbleBounds.width / 2),
      bubbleBounds.top + (bubbleBounds.height / 2),
    );

    final currentDisplay = displays.firstWhere(
      (display) => _displayVisibleRect(display).contains(bubbleCenter),
      orElse: () => primaryDisplay,
    );

    final visibleRect = _displayVisibleRect(currentDisplay);
    final preferredLeft = bubbleBounds.right - targetSize.width;
    final preferredTop = bubbleBounds.top - 16;

    final left =
        preferredLeft.clamp(visibleRect.left, visibleRect.right - targetSize.width);
    final top =
        preferredTop.clamp(visibleRect.top, visibleRect.bottom - targetSize.height);

    return Rect.fromLTWH(left, top, targetSize.width, targetSize.height);
  }

  Future<Rect> _currentDisplayVisibleRect(Rect anchor) async {
    final displays = await screenRetriever.getAllDisplays();
    final primaryDisplay = await screenRetriever.getPrimaryDisplay();
    final center = Offset(
      anchor.left + (anchor.width / 2),
      anchor.top + (anchor.height / 2),
    );
    final display = displays.firstWhere(
      (item) => _displayVisibleRect(item).contains(center),
      orElse: () => primaryDisplay,
    );
    return _displayVisibleRect(display);
  }

  Rect _displayVisibleRect(Display display) {
    final visiblePosition = display.visiblePosition ?? Offset.zero;
    final visibleSize = display.visibleSize ?? display.size;
    return Rect.fromLTWH(
      visiblePosition.dx,
      visiblePosition.dy,
      visibleSize.width,
      visibleSize.height,
    );
  }

  Rect _clampRectToVisibleArea(Rect visibleRect, Rect rect) {
    final width = rect.width.clamp(0.0, visibleRect.width);
    final height = rect.height.clamp(0.0, visibleRect.height);
    final left = rect.left.clamp(visibleRect.left, visibleRect.right - width);
    final top = rect.top.clamp(visibleRect.top, visibleRect.bottom - height);
    return Rect.fromLTWH(left, top, width, height);
  }

  Future<void> _animateOpacity({
    required double from,
    required double to,
  }) async {
    if (!isDesktop) {
      return;
    }
    const steps = 4;
    final delta = (to - from) / steps;
    for (var i = 0; i <= steps; i++) {
      final value = (from + delta * i).clamp(0.0, 1.0);
      await windowManager.setOpacity(value);
      if (i < steps) {
        await Future<void>.delayed(
          Duration(
            milliseconds: (_transitionDuration.inMilliseconds / steps).round(),
          ),
        );
      }
    }
  }
}
