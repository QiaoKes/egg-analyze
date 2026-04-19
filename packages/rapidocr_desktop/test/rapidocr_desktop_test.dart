import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:rapidocr_desktop/rapidocr_desktop.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('reuses a long-lived worker and maps OCR lines', () async {
    final runtimeRoot = await Directory.systemTemp.createTemp(
      'rapidocr-desktop-test-',
    );

    const fakeWorker = r'''
import json
import sys

counter = 0

print(json.dumps({"ready": True}), flush=True)
for raw in sys.stdin:
    raw = raw.strip()
    if not raw:
        continue
    counter += 1
    print(
        json.dumps(
            {
                "ok": True,
                "lines": [
                    {
                        "text": f"0.{counter}23",
                        "left": 12,
                        "top": 34,
                        "width": 56,
                        "height": 18,
                    }
                ],
            }
        ),
        flush=True,
    )
''';

    final client = RapidOcrDesktop(
      pythonExecutable: 'python3',
      workerScript: fakeWorker,
      runtimeRoot: runtimeRoot,
    );

    try {
      final source = img.Image(width: 520, height: 280);
      final bytes = Uint8List.fromList(img.encodePng(source));

      final first = await client.recognize(bytes);
      final second = await client.recognize(bytes);

      expect(first.lines, hasLength(1));
      expect(first.lines.first.text, '0.123');
      expect(first.lines.first.bounds.left, 12);
      expect(first.lines.first.bounds.top, 34);

      expect(second.lines, hasLength(1));
      expect(second.lines.first.text, '0.223');
    } finally {
      await client.dispose();
      if (await runtimeRoot.exists()) {
        await runtimeRoot.delete(recursive: true);
      }
    }
  });

  test('surfaces a helpful startup error when RapidOCR is unavailable', () async {
    final runtimeRoot = await Directory.systemTemp.createTemp(
      'rapidocr-desktop-error-',
    );

    const failingWorker = r'''
import json
print(json.dumps({"ready": False, "error": "rapidocr missing"}), flush=True)
''';

    final client = RapidOcrDesktop(
      pythonExecutable: 'python3',
      workerScript: failingWorker,
      runtimeRoot: runtimeRoot,
    );

    try {
      final bytes = Uint8List.fromList(img.encodePng(img.Image(width: 16, height: 16)));
      await expectLater(
        client.recognize(bytes),
        throwsA(
          isA<RapidOcrDesktopException>().having(
            (error) => error.toString(),
            'message',
            contains('rapidocr missing'),
          ),
        ),
      );
    } finally {
      await client.dispose();
      if (await runtimeRoot.exists()) {
        await runtimeRoot.delete(recursive: true);
      }
    }
  });
}
