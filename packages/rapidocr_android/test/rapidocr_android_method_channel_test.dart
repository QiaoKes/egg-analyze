import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:rapidocr_android/rapidocr_android.dart';
import 'package:rapidocr_android/rapidocr_android_method_channel.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final platform = MethodChannelRapidocrAndroid();
  const channel = MethodChannel('rapidocr_android/methods');

  setUp(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (methodCall) async {
      switch (methodCall.method) {
        case 'initialize':
        case 'dispose':
          return null;
        case 'recognize':
          return <String, Object?>{
            'lines': <Map<String, Object?>>[
              <String, Object?>{
                'text': '神奇的蛋',
                'left': 10.0,
                'top': 20.0,
                'width': 30.0,
                'height': 40.0,
              },
            ],
          };
        case 'recognizeBatch':
          return <Map<String, Object?>>[
            <String, Object?>{
              'lines': <Map<String, Object?>>[
                <String, Object?>{
                  'text': '0.23',
                  'left': 1.0,
                  'top': 2.0,
                  'width': 3.0,
                  'height': 4.0,
                },
              ],
            },
          ];
      }
      return null;
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('recognize maps channel payload to ocr document', () async {
    final result = await platform.recognize(
      Uint8List(0),
      mode: RapidOcrMode.full,
    );

    expect(result.lines.single.text, '神奇的蛋');
    expect(result.lines.single.bounds.left, 10);
  });

  test('recognizeBatch maps channel payload to ocr documents', () async {
    final result = await platform.recognizeBatch(
      [Uint8List(0)],
      mode: RapidOcrMode.recOnly,
    );

    expect(result, hasLength(1));
    expect(result.single.lines.single.text, '0.23');
  });
}
