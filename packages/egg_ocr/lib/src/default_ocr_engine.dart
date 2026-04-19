import 'dart:io';

import 'package:egg_core/egg_core.dart';

import 'mlkit_ocr_engine.dart';
import 'platform_ocr_engine.dart';
import 'rapidocr_desktop_engine.dart';

class DefaultOcrEngine {
  static OcrEngine create() {
    if (Platform.isMacOS || Platform.isWindows) {
      return RapidOcrDesktopEngine();
    }
    if (Platform.isAndroid) {
      return MlKitOcrEngine();
    }
    throw UnsupportedError('Unsupported OCR platform');
  }

  static Future<void> dispose(OcrEngine engine) async {
    if (engine is RapidOcrDesktopEngine) {
      await engine.dispose();
      return;
    }
    if (engine is MlKitOcrEngine) {
      await engine.dispose();
      return;
    }
    if (engine is PlatformOcrEngine) {
      engine.dispose();
    }
  }
}
