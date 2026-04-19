import 'dart:async';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:egg_core/egg_core.dart';
import 'package:platform_ocr/platform_ocr.dart' as platform_ocr;

class PlatformOcrEngine implements OcrEngine {
  PlatformOcrEngine({platform_ocr.PlatformOcr? platformOcr})
      : _platformOcr = platformOcr ?? platform_ocr.PlatformOcr();

  final platform_ocr.PlatformOcr _platformOcr;

  @override
  Future<OcrDocument> recognize(Uint8List imageBytes) async {
    final imageSize = await _decodeImageSize(imageBytes);
    final result = await _platformOcr
        .recognizeText(platform_ocr.OcrSource.memory(imageBytes));
    final lines = result.lines
        .where((value) => value.text.trim().isNotEmpty)
        .map(
          (value) => OcrLine(
            text: value.text.trim(),
            bounds: _normalizeBoundingBox(
              value.boundingBox,
              imageSize: imageSize,
            ),
          ),
        )
        .toList(growable: false);
    return OcrDocument(lines: lines);
  }

  Future<ui.Size> _decodeImageSize(Uint8List imageBytes) {
    final completer = Completer<ui.Size>();
    ui.decodeImageFromList(imageBytes, (image) {
      completer.complete(
        ui.Size(image.width.toDouble(), image.height.toDouble()),
      );
    });
    return completer.future;
  }

  ui.Rect _normalizeBoundingBox(
    platform_ocr.Rect rect, {
    required ui.Size imageSize,
  }) {
    final looksNormalized = rect.left >= 0 &&
        rect.top >= 0 &&
        rect.width >= 0 &&
        rect.height >= 0 &&
        rect.left <= 1 &&
        rect.top <= 1 &&
        rect.width <= 1 &&
        rect.height <= 1;

    if (!looksNormalized) {
      return ui.Rect.fromLTWH(
        rect.left,
        rect.top,
        rect.width,
        rect.height,
      );
    }

    return ui.Rect.fromLTWH(
      rect.left * imageSize.width,
      rect.top * imageSize.height,
      rect.width * imageSize.width,
      rect.height * imageSize.height,
    );
  }

  void dispose() {
    _platformOcr.dispose();
  }
}
