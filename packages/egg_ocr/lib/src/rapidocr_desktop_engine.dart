import 'dart:typed_data';

import 'package:egg_core/egg_core.dart';
import 'package:rapidocr_desktop/rapidocr_desktop.dart';

class RapidOcrDesktopEngine implements OcrEngine {
  RapidOcrDesktopEngine({RapidOcrDesktop? client})
      : _client = client ?? RapidOcrDesktop();

  final RapidOcrDesktop _client;

  @override
  Future<OcrDocument> recognize(Uint8List imageBytes) {
    return _client.recognize(imageBytes);
  }

  Future<void> dispose() => _client.dispose();
}

