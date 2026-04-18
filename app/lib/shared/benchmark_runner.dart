import 'dart:convert';
import 'package:egg_core/egg_core.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'providers.dart';

final benchmarkRunnerProvider = Provider<OcrBenchmarkRunner>((ref) {
  return OcrBenchmarkRunner(ref);
});

class OcrBenchmarkRunner {
  OcrBenchmarkRunner(this.ref);

  final Ref ref;

  Future<OcrBenchmarkReport> run() async {
    final manifest = await _loadCasesFromManifest(
      'assets/benchmarks/benchmarks.json',
    );

    final dataset = await ref.read(datasetRepositoryProvider).load();
    final engine = MatchingEngine(dataset.dataset);
    final extractor = const MeasurementExtractor();
    final ocr = ref.read(ocrEngineProvider);

    final cases = <OcrBenchmarkCaseResult>[];
    for (final item in manifest) {
      final bytes =
          (await rootBundle.load(item.assetPath)).buffer.asUint8List();
      final document = await ocr.recognize(bytes);
      final measurements = await extractor.extractBestMeasurements(
        lines: document.lines,
        priors: engine.measurementPriors,
        sourceBytes: bytes,
        recognizeCrop: ocr.recognize,
      );

      final entries = measurements
          .map(
            (measurement) => AnalysisEntry(
              measurement: measurement,
              candidates: engine.search(
                heightInCentimeters: measurement.heightInMeters * 100,
                weightInKg: measurement.weightInKg,
              ),
            ),
          )
          .toList(growable: false);

      final passed = _matchesExpected(item.expectedMeasurements, measurements);
      cases.add(
        OcrBenchmarkCaseResult(
          config: item,
          ocrDocument: document,
          measurements: measurements,
          entries: entries,
          passed: passed,
        ),
      );
    }

    final passedCount = cases.where((item) => item.passed).length;
    return OcrBenchmarkReport(
      generatedAt: DateTime.now(),
      cases: cases,
      passedCount: passedCount,
      totalCount: cases.length,
    );
  }

  bool _matchesExpected(
    List<ExpectedMeasurement> expected,
    List<Measurement> actual,
  ) {
    if (expected.length != actual.length) {
      return false;
    }

    final left = [...expected]..sort(
        (a, b) => '${a.size}-${a.weight}'.compareTo('${b.size}-${b.weight}'));
    final right = [...actual]..sort((a, b) =>
        '${a.heightInMeters}-${a.weightInKg}'
            .compareTo('${b.heightInMeters}-${b.weightInKg}'));

    for (var i = 0; i < left.length; i++) {
      if ((left[i].size - right[i].heightInMeters).abs() > 0.001) {
        return false;
      }
      if ((left[i].weight - right[i].weightInKg).abs() > 0.001) {
        return false;
      }
    }
    return true;
  }

  Future<List<BenchmarkCase>> _loadCasesFromManifest(
    String assetManifestPath,
  ) async {
    final raw = await rootBundle.loadString(assetManifestPath);
    return (jsonDecode(raw) as List<dynamic>)
        .whereType<Map<String, dynamic>>()
        .map(BenchmarkCase.fromJson)
        .toList(growable: false);
  }
}

class OcrBenchmarkReport {
  const OcrBenchmarkReport({
    required this.generatedAt,
    required this.cases,
    required this.passedCount,
    required this.totalCount,
  });

  final DateTime generatedAt;
  final List<OcrBenchmarkCaseResult> cases;
  final int passedCount;
  final int totalCount;
}

class OcrBenchmarkCaseResult {
  const OcrBenchmarkCaseResult({
    required this.config,
    required this.ocrDocument,
    required this.measurements,
    required this.entries,
    required this.passed,
  });

  final BenchmarkCase config;
  final OcrDocument ocrDocument;
  final List<Measurement> measurements;
  final List<AnalysisEntry> entries;
  final bool passed;
}

class BenchmarkCase {
  const BenchmarkCase({
    required this.assetPath,
    required this.caseName,
    required this.expectedMeasurements,
    required this.sourceGroup,
  });

  final String assetPath;
  final String caseName;
  final List<ExpectedMeasurement> expectedMeasurements;
  final String sourceGroup;

  factory BenchmarkCase.fromJson(
    Map<String, dynamic> json,
  ) {
    final expected = ((json['measurements'] as List<dynamic>?) ?? const [])
        .whereType<Map<String, dynamic>>()
        .map(ExpectedMeasurement.fromJson)
        .toList(growable: false);

    return BenchmarkCase(
      assetPath: json['image'] as String,
      caseName: json['id'] as String,
      expectedMeasurements: expected,
      sourceGroup: json['group'] as String,
    );
  }
}

class ExpectedMeasurement {
  const ExpectedMeasurement({
    required this.size,
    required this.weight,
  });

  final double size;
  final double weight;

  factory ExpectedMeasurement.fromJson(Map<String, dynamic> json) {
    double toDouble(Object? value) => (value as num).toDouble();
    return ExpectedMeasurement(
      size: toDouble(json['size']),
      weight: toDouble(json['weight']),
    );
  }
}
