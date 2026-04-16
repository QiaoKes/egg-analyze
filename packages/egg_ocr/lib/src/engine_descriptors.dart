import 'package:flutter/foundation.dart';

String currentOcrEngineLabel() {
  return switch (defaultTargetPlatform) {
    TargetPlatform.macOS => 'platform_ocr (Vision.framework)',
    TargetPlatform.windows => 'platform_ocr (Windows.Media.Ocr)',
    TargetPlatform.android => 'google_mlkit_text_recognition',
    _ => 'unsupported',
  };
}
