import 'dart:typed_data';
import 'dart:ui';

import 'package:egg_core/egg_core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:plugin_platform_interface/plugin_platform_interface.dart';
import 'package:rapidocr_android/rapidocr_android.dart';
import 'package:rapidocr_android/rapidocr_android_method_channel.dart';
import 'package:rapidocr_android/rapidocr_android_platform_interface.dart';

class MockRapidocrAndroidPlatform
    with MockPlatformInterfaceMixin
    implements RapidocrAndroidPlatform {
  @override
  Future<void> dispose() async {}

  @override
  Future<void> initialize() async {}

  @override
  Future<OcrDocument> recognize(
    Uint8List imageBytes, {
    required RapidOcrMode mode,
  }) async {
    return OcrDocument(
      lines: [
        OcrLine(
          text: mode.value,
          bounds: Rect.zero,
        ),
      ],
    );
  }

  @override
  Future<List<OcrDocument>> recognizeBatch(
    List<Uint8List> imageBytesList, {
    required RapidOcrMode mode,
  }) async {
    return imageBytesList
        .map(
          (_) => OcrDocument(
            lines: [
              OcrLine(text: mode.value, bounds: Rect.zero),
            ],
          ),
        )
        .toList(growable: false)
        .cast<OcrDocument>();
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final initialPlatform = RapidocrAndroidPlatform.instance;

  test('$MethodChannelRapidocrAndroid is the default instance', () {
    expect(initialPlatform, isInstanceOf<MethodChannelRapidocrAndroid>());
  });

  test('recognize proxies to platform implementation', () async {
    RapidocrAndroidPlatform.instance = MockRapidocrAndroidPlatform();
    final plugin = RapidOcrAndroid();

    final result = await plugin.recognize(
      Uint8List(0),
      mode: RapidOcrMode.recOnly,
    );

    expect(result.lines.single.text, 'rec_only');
  });
}
