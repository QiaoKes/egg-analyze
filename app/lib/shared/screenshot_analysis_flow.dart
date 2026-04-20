import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';
import 'package:screen_capturer/screen_capturer.dart';
import 'package:window_manager/window_manager.dart';

import 'providers.dart';

bool get supportsRegionCapture => Platform.isMacOS || Platform.isWindows;

Future<void> captureRegionAndAnalyze(
  BuildContext context,
  WidgetRef ref, {
  Future<void> Function()? onSuccess,
  VoidCallback? onViewDetails,
}) async {
  if (!supportsRegionCapture) {
    return;
  }

  final shouldHideWindow = Platform.isMacOS || Platform.isWindows;
  try {
    if (Platform.isMacOS) {
      final allowed = await screenCapturer.isAccessAllowed();
      if (!allowed) {
        await screenCapturer.requestAccess();
        if (!context.mounted) {
          return;
        }
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('请先在系统设置中允许屏幕录制权限，然后再次点击截屏分析。'),
          ),
        );
        return;
      }
    }

    if (shouldHideWindow) {
      await windowManager.hide();
      await Future<void>.delayed(const Duration(milliseconds: 160));
    }

    final capturedBytes = await _captureRegionImageBytes();
    if (capturedBytes == null) {
      if (!context.mounted) {
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('没有拿到截图内容，可能是取消了截图，或 Windows 截图结果还没写回应用。请重试一次。'),
        ),
      );
      return;
    }

    final result = await ref.read(analysisControllerProvider).analyzeBytes(
          capturedBytes,
          label: '区域截图',
        );
    if (!context.mounted) {
      return;
    }

    if (result.entries.isEmpty) {
      final message = result.ocrDocument.lines.isEmpty
          ? '没有识别到有效文字，请尽量截全单个蛋的信息区域后重试。'
          : '没有提取到有效的蛋尺寸/蛋重量，请重新截图或截得更完整一些。';
      final messenger = ScaffoldMessenger.of(context);
      messenger.hideCurrentSnackBar();
      messenger.showSnackBar(
        SnackBar(
          content: Text(message),
          action: onViewDetails == null
              ? null
              : SnackBarAction(
                  label: '查看详情',
                  onPressed: onViewDetails,
                ),
        ),
      );
      return;
    }

    if (onSuccess != null) {
      await onSuccess();
    }
  } catch (error) {
    if (!context.mounted) {
      return;
    }
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('截屏分析失败：$error')),
    );
  } finally {
    if (shouldHideWindow) {
      await windowManager.show();
      await windowManager.focus();
    }
  }
}

Future<Uint8List?> _captureRegionImageBytes() async {
  String? imagePath;
  try {
    if (Platform.isWindows) {
      final tempDir = await getTemporaryDirectory();
      imagePath =
          '${tempDir.path}${Platform.pathSeparator}egg-analyze-capture-${DateTime.now().microsecondsSinceEpoch}.png';
    }

    final captured = await screenCapturer.capture(
      mode: CaptureMode.region,
      imagePath: imagePath,
      // Windows package relies on Snipping Tool + clipboard. Reading the
      // clipboard back is flaky there, so we persist to a temp file instead.
      copyToClipboard: !Platform.isWindows,
      silent: true,
    );
    if (captured?.imageBytes != null) {
      return captured!.imageBytes!;
    }

    if (imagePath == null) {
      return null;
    }

    final imageFile = File(imagePath);
    if (await imageFile.exists()) {
      return await imageFile.readAsBytes();
    }
    return null;
  } finally {
    if (imagePath != null) {
      final imageFile = File(imagePath);
      if (await imageFile.exists()) {
        await imageFile.delete();
      }
    }
  }
}
