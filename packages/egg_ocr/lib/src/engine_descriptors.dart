import 'package:flutter/foundation.dart';

String currentOcrEngineLabel() {
  return switch (defaultTargetPlatform) {
    TargetPlatform.macOS => 'RapidOCR (Python worker + onnxruntime)',
    TargetPlatform.windows => 'RapidOCR (Python worker + onnxruntime)',
    TargetPlatform.android => 'RapidOCR (Android ONNX)',
    _ => 'unsupported',
  };
}
