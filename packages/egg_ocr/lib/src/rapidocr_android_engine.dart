import 'dart:typed_data';

import 'package:egg_core/egg_core.dart';
import 'package:rapidocr_android/rapidocr_android.dart';

class RapidOcrAndroidEngine implements BatchTextOnlyOcrEngine {
  RapidOcrAndroidEngine({RapidOcrAndroid? client})
      : _client = client ?? RapidOcrAndroid();

  final RapidOcrAndroid _client;
  Future<void>? _initialization;

  Future<void> _ensureInitialized() {
    return _initialization ??= _client.initialize();
  }

  @override
  Future<OcrDocument> recognize(Uint8List imageBytes) async {
    await _ensureInitialized();
    return _client.recognize(
      imageBytes,
      mode: RapidOcrMode.full,
    );
  }

  @override
  Future<OcrDocument> recognizeTextOnly(Uint8List imageBytes) async {
    await _ensureInitialized();
    return _client.recognize(
      imageBytes,
      mode: RapidOcrMode.recOnly,
    );
  }

  @override
  Future<List<OcrDocument>> recognizeTextOnlyBatch(
    List<Uint8List> imageBytesList,
  ) async {
    await _ensureInitialized();
    return _client.recognizeBatch(
      imageBytesList,
      mode: RapidOcrMode.recOnly,
    );
  }

  Future<void> dispose() async {
    await _initialization;
    await _client.dispose();
    _initialization = null;
  }
}
