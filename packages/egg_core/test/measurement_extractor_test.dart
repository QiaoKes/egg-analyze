import 'dart:ui';

import 'package:egg_core/egg_core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const extractor = MeasurementExtractor();
  const priors = MeasurementPriors(
    diameter: Range(min: 0.03, max: 1.2),
    weight: Range(min: 0.03, max: 300),
  );

  test('extracts measurement from same row values', () {
    final result = extractor.extractMeasurements([
      const OcrLine(text: '0.21', bounds: Rect.fromLTWH(80, 120, 60, 20)),
      const OcrLine(text: '6.260', bounds: Rect.fromLTWH(180, 120, 80, 20)),
    ], priors);

    expect(result, hasLength(1));
    expect(result.first.heightInMeters, closeTo(0.21, 0.001));
    expect(result.first.weightInKg, closeTo(6.26, 0.001));
  });

  test('ignores percent and date noise', () {
    final result = extractor.extractMeasurements([
      const OcrLine(text: '2026-04-17', bounds: Rect.fromLTWH(0, 0, 120, 20)),
      const OcrLine(text: '55%', bounds: Rect.fromLTWH(0, 40, 40, 20)),
      const OcrLine(text: '0.23', bounds: Rect.fromLTWH(60, 80, 50, 20)),
      const OcrLine(text: '2.750', bounds: Rect.fromLTWH(60, 120, 80, 20)),
    ], priors);

    expect(result, hasLength(1));
    expect(result.first.heightInMeters, closeTo(0.23, 0.001));
    expect(result.first.weightInKg, closeTo(2.75, 0.001));
  });

  test('prefers full number over embedded decimal substring', () {
    final result = extractor.extractMeasurements([
      const OcrLine(text: '0.25<×', bounds: Rect.fromLTWH(80, 120, 70, 20)),
      const OcrLine(text: '10.069A', bounds: Rect.fromLTWH(80, 160, 90, 20)),
    ], priors);

    expect(result, hasLength(1));
    expect(result.first.heightInMeters, closeTo(0.25, 0.001));
    expect(result.first.weightInKg, closeTo(10.069, 0.001));
  });

  test('captures egg name from title line above measurement', () {
    final result = extractor.extractMeasurements([
      const OcrLine(text: '孵化装置1', bounds: Rect.fromLTWH(10, 20, 90, 24)),
      const OcrLine(text: '神奇的蛋', bounds: Rect.fromLTWH(120, 30, 120, 24)),
      const OcrLine(text: '完成', bounds: Rect.fromLTWH(150, 70, 50, 20)),
      const OcrLine(text: '0.23', bounds: Rect.fromLTWH(150, 120, 60, 20)),
      const OcrLine(text: '2.750', bounds: Rect.fromLTWH(150, 160, 80, 20)),
    ], priors);

    expect(result, hasLength(1));
    expect(result.first.eggName, '神奇的蛋');
  });
}
