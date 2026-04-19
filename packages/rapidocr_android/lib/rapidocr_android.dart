import 'dart:typed_data';

import 'package:egg_core/egg_core.dart';

import 'rapidocr_android_platform_interface.dart';

enum RapidOcrMode {
  full('full'),
  recOnly('rec_only');

  const RapidOcrMode(this.value);

  final String value;
}

class RapidOcrAndroidException implements Exception {
  const RapidOcrAndroidException(this.message);

  final String message;

  @override
  String toString() => 'RapidOcrAndroidException: $message';
}

class RapidOcrAndroid {
  RapidOcrAndroid({RapidocrAndroidPlatform? platform})
      : _platform = platform ?? RapidocrAndroidPlatform.instance;

  final RapidocrAndroidPlatform _platform;

  Future<void> initialize() => _platform.initialize();

  Future<OcrDocument> recognize(
    Uint8List imageBytes, {
    RapidOcrMode mode = RapidOcrMode.full,
  }) {
    return _platform.recognize(
      imageBytes,
      mode: mode,
    );
  }

  Future<List<OcrDocument>> recognizeBatch(
    List<Uint8List> imageBytesList, {
    RapidOcrMode mode = RapidOcrMode.recOnly,
  }) {
    return _platform.recognizeBatch(
      imageBytesList,
      mode: mode,
    );
  }

  Future<void> dispose() => _platform.dispose();
}
