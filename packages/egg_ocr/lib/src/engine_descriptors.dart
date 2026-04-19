import 'package:flutter/foundation.dart';

String currentOcrEngineLabel() {
  return switch (defaultTargetPlatform) {
    TargetPlatform.macOS => 'RapidOCR (Python worker + onnxruntime)',
    TargetPlatform.windows => 'RapidOCR (Python worker + onnxruntime)',
    TargetPlatform.android => 'google_mlkit_text_recognition',
    _ => 'unsupported',
  };
}
