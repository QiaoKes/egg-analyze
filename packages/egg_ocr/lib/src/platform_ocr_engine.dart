import 'dart:typed_data';
import 'dart:ui';

import 'package:egg_core/egg_core.dart';
import 'package:platform_ocr/platform_ocr.dart' as platform_ocr;

class PlatformOcrEngine implements OcrEngine {
  PlatformOcrEngine({platform_ocr.PlatformOcr? platformOcr})
      : _platformOcr = platformOcr ?? platform_ocr.PlatformOcr();

  final platform_ocr.PlatformOcr _platformOcr;

  @override
  Future<OcrDocument> recognize(Uint8List imageBytes) async {
    final result = await _platformOcr
        .recognizeText(platform_ocr.OcrSource.memory(imageBytes));
    final lines = result.lines
        .where((value) => value.text.trim().isNotEmpty)
        .map(
          (value) => OcrLine(
            text: value.text.trim(),
            bounds: Rect.fromLTWH(
              value.boundingBox.left,
              value.boundingBox.top,
              value.boundingBox.width,
              value.boundingBox.height,
            ),
          ),
        )
        .toList(growable: false);
    return OcrDocument(lines: lines);
  }
}
