import 'dart:io';

import 'package:egg_core/egg_core.dart';

import 'mlkit_ocr_engine.dart';
import 'platform_ocr_engine.dart';

class DefaultOcrEngine {
  static OcrEngine create() {
    if (Platform.isMacOS || Platform.isWindows) {
      return PlatformOcrEngine();
    }
    if (Platform.isAndroid) {
      return MlKitOcrEngine();
    }
    throw UnsupportedError('Unsupported OCR platform');
  }
}
