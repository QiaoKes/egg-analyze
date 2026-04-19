import 'dart:typed_data';

import 'package:egg_core/egg_core.dart';
import 'package:rapidocr_desktop/rapidocr_desktop.dart';

class RapidOcrDesktopEngine implements BatchTextOnlyOcrEngine {
  RapidOcrDesktopEngine({RapidOcrDesktop? client})
      : _client = client ?? RapidOcrDesktop();

  final RapidOcrDesktop _client;

  @override
  Future<OcrDocument> recognize(Uint8List imageBytes) {
    return _client.recognize(imageBytes);
  }

  @override
  Future<OcrDocument> recognizeTextOnly(Uint8List imageBytes) {
    return _client.recognizeTextOnly(imageBytes);
  }

  @override
  Future<List<OcrDocument>> recognizeTextOnlyBatch(
    List<Uint8List> imageBytesList,
  ) {
    return _client.recognizeTextOnlyBatch(imageBytesList);
  }

  Future<void> dispose() => _client.dispose();
}
