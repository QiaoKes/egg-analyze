import 'dart:io';
import 'dart:typed_data';

import 'package:egg_core/egg_core.dart';
import 'package:google_mlkit_text_recognition/google_mlkit_text_recognition.dart';
import 'package:image/image.dart' as img;
import 'package:path_provider/path_provider.dart';

class MlKitOcrEngine implements OcrEngine {
  static const _minimumImageEdge = 32;

  MlKitOcrEngine({
    TextRecognizer? recognizer,
  }) : _recognizer =
            recognizer ?? TextRecognizer(script: TextRecognitionScript.latin);

  final TextRecognizer _recognizer;

  @override
  Future<OcrDocument> recognize(Uint8List imageBytes) async {
    final preparedBytes = _ensureMinimumSize(imageBytes);
    final primary = await _recognizePreparedBytes(preparedBytes);
    if (!_needsUpscaledRetry(preparedBytes, primary)) {
      return primary;
    }

    final upscaledBytes = _upscaleForRetry(preparedBytes);
    if (upscaledBytes == null) {
      return primary;
    }

    final retried = await _recognizePreparedBytes(upscaledBytes);
    return _scoreDocument(retried) >= _scoreDocument(primary) ? retried : primary;
  }

  Uint8List _ensureMinimumSize(Uint8List imageBytes) {
    final decoded = img.decodeImage(imageBytes);
    if (decoded == null) {
      return imageBytes;
    }

    if (decoded.width >= _minimumImageEdge &&
        decoded.height >= _minimumImageEdge) {
      return imageBytes;
    }

    final scale = [
      _minimumImageEdge / decoded.width,
      _minimumImageEdge / decoded.height,
    ].reduce((a, b) => a > b ? a : b);

    final resized = img.copyResize(
      decoded,
      width: (decoded.width * scale).ceil(),
      height: (decoded.height * scale).ceil(),
      interpolation: img.Interpolation.nearest,
    );
    return Uint8List.fromList(img.encodePng(resized));
  }

  Future<OcrDocument> _recognizePreparedBytes(Uint8List imageBytes) async {
    final tempDir = await getTemporaryDirectory();
    final file = File(
      '${tempDir.path}/egg-analyze-ocr-${DateTime.now().microsecondsSinceEpoch}.png',
    );
    await file.writeAsBytes(imageBytes, flush: true);
    try {
      final inputImage = InputImage.fromFilePath(file.path);
      final recognizedText = await _recognizer.processImage(inputImage);
      final lines = <OcrLine>[];
      for (final block in recognizedText.blocks) {
        for (final line in block.lines) {
          final text = _normalizeMlKitText(line.text);
          if (text.trim().isEmpty) {
            continue;
          }
          lines.add(
            OcrLine(
              text: text,
              bounds: line.boundingBox,
            ),
          );
        }
      }
      return OcrDocument(lines: lines);
    } finally {
      if (await file.exists()) {
        await file.delete();
      }
    }
  }

  bool _needsUpscaledRetry(Uint8List imageBytes, OcrDocument document) {
    final decoded = img.decodeImage(imageBytes);
    if (decoded == null) {
      return false;
    }
    if (decoded.width >= 1400 || decoded.height >= 1400) {
      return false;
    }
    final decimalLikeLines = document.lines
        .where((line) => RegExp(r'\d\.\d').hasMatch(line.text))
        .length;
    return decimalLikeLines < 2;
  }

  Uint8List? _upscaleForRetry(Uint8List imageBytes) {
    final decoded = img.decodeImage(imageBytes);
    if (decoded == null) {
      return null;
    }
    final resized = img.copyResize(
      decoded,
      width: decoded.width * 2,
      height: decoded.height * 2,
      interpolation: img.Interpolation.cubic,
    );
    return Uint8List.fromList(img.encodePng(resized));
  }

  int _scoreDocument(OcrDocument document) {
    var score = 0;
    for (final line in document.lines) {
      if (RegExp(r'\d').hasMatch(line.text)) {
        score += 1;
      }
      if (RegExp(r'\d\.\d').hasMatch(line.text)) {
        score += 3;
      }
      if (line.text.length >= 4) {
        score += 1;
      }
    }
    return score;
  }

  String _normalizeMlKitText(String input) {
    final runes = input.runes.toList();
    for (var i = 0; i < runes.length; i++) {
      final mapped = _mapMlKitDigitLikeRune(runes[i]);
      if (mapped == null) {
        continue;
      }
      if (!_isNumericContextRune(runes, i)) {
        continue;
      }
      runes[i] = mapped;
    }
    return String.fromCharCodes(runes);
  }

  int? _mapMlKitDigitLikeRune(int rune) {
    switch (String.fromCharCode(rune)) {
      case 'e':
      case 'E':
        return '5'.codeUnitAt(0);
      case 'B':
      case 'b':
        return '3'.codeUnitAt(0);
      default:
        return null;
    }
  }

  bool _isNumericContextRune(List<int> values, int index) {
    final leftNumeric = index > 0 && _isDigitOrDot(values[index - 1]);
    final rightNumeric =
        index + 1 < values.length && _isDigitOrDot(values[index + 1]);
    return leftNumeric || rightNumeric;
  }

  bool _isDigitOrDot(int rune) {
    final value = String.fromCharCode(rune);
    return RegExp(r'[0-9.]').hasMatch(value);
  }

  Future<void> dispose() {
    return _recognizer.close();
  }
}
