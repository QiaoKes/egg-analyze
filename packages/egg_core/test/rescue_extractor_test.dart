import 'dart:typed_data';
import 'dart:ui';

import 'package:egg_core/egg_core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

void main() {
  const extractor = MeasurementExtractor();
  const priors = MeasurementPriors(
    diameter: Range(min: 0.04, max: 1.1),
    weight: Range(min: 0.03, max: 280),
  );

  test('rescues broken weight line using crop OCR callback', () async {
    final source = img.Image(width: 400, height: 500);
    final bytes = Uint8List.fromList(img.encodePng(source));

    final measurements = await extractor.extractBestMeasurements(
      lines: const [
        OcrLine(text: '0.23<×', bounds: Rect.fromLTWH(120, 120, 80, 24)),
        OcrLine(text: '2.75A', bounds: Rect.fromLTWH(120, 160, 80, 24)),
        OcrLine(text: '0.21<×', bounds: Rect.fromLTWH(120, 300, 80, 24)),
        OcrLine(text: 'M60E\'0', bounds: Rect.fromLTWH(120, 340, 90, 24)),
      ],
      priors: priors,
      sourceBytes: bytes,
      recognizeCrop: (_) async => const OcrDocument(
        lines: [
          OcrLine(text: '0.309A', bounds: Rect.zero),
        ],
      ),
    );

    expect(measurements, hasLength(2));
    expect(
      measurements.any(
        (m) =>
            (m.heightInMeters - 0.21).abs() < 0.001 &&
            (m.weightInKg - 0.309).abs() < 0.001,
      ),
      isTrue,
    );
  });
}
