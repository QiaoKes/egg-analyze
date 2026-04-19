import 'package:egg_core/egg_core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

import 'rapidocr_android.dart';
import 'rapidocr_android_platform_interface.dart';

class MethodChannelRapidocrAndroid extends RapidocrAndroidPlatform {
  @visibleForTesting
  final methodChannel = const MethodChannel('rapidocr_android/methods');

  @override
  Future<void> initialize() async {
    await methodChannel.invokeMethod<void>('initialize');
  }

  @override
  Future<OcrDocument> recognize(
    Uint8List imageBytes, {
    required RapidOcrMode mode,
  }) async {
    final response = await methodChannel.invokeMapMethod<String, Object?>(
      'recognize',
      <String, Object?>{
        'imageBytes': imageBytes,
        'mode': mode.value,
      },
    );
    return _documentFromMap(response);
  }

  @override
  Future<List<OcrDocument>> recognizeBatch(
    List<Uint8List> imageBytesList, {
    required RapidOcrMode mode,
  }) async {
    final response = await methodChannel.invokeMethod<List<Object?>>(
      'recognizeBatch',
      <String, Object?>{
        'images': imageBytesList,
        'mode': mode.value,
      },
    );
    final items = response ?? const <Object?>[];
    return items
        .whereType<Map<Object?, Object?>>()
        .map(
          (item) => _documentFromMap(
            item.map(
              (key, value) => MapEntry(
                key.toString(),
                value,
              ),
            ),
          ),
        )
        .toList(growable: false);
  }

  @override
  Future<void> dispose() async {
    await methodChannel.invokeMethod<void>('dispose');
  }

  OcrDocument _documentFromMap(Map<String, Object?>? map) {
    final lines = (map?['lines'] as List<Object?>? ?? const <Object?>[])
        .whereType<Map<Object?, Object?>>()
        .map((item) {
          final json = item.map(
            (key, value) => MapEntry(
              key.toString(),
              value,
            ),
          );
          double toDouble(Object? value) {
            if (value is num) {
              return value.toDouble();
            }
            return double.tryParse('$value') ?? 0;
          }

          return OcrLine(
            text: (json['text'] as String? ?? '').trim(),
            bounds: Rect.fromLTWH(
              toDouble(json['left']),
              toDouble(json['top']),
              toDouble(json['width']),
              toDouble(json['height']),
            ),
          );
        })
        .where((line) => line.text.isNotEmpty)
        .toList(growable: false);
    return OcrDocument(lines: lines);
  }
}
