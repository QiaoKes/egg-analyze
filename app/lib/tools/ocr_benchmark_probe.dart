import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';

import '../shared/benchmark_runner.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final container = ProviderContainer();
  try {
    final report = await container.read(benchmarkRunnerProvider).run();

    for (final item in report.cases) {
      debugPrint(
        'CASE ${item.config.caseName}: ${item.passed ? 'PASS' : 'FAIL'}',
      );
      debugPrint('  group: ${item.config.sourceGroup}');
      debugPrint('  ocr_lines: ${item.ocrDocument.lines.length}');
      for (final line in item.ocrDocument.lines.take(8)) {
        debugPrint(
          '    - ${line.text} @ '
          '(${line.bounds.left.toStringAsFixed(1)}, '
          '${line.bounds.top.toStringAsFixed(1)}, '
          '${line.bounds.width.toStringAsFixed(1)}, '
          '${line.bounds.height.toStringAsFixed(1)})',
        );
      }
      debugPrint(
        '  expected: ${item.config.expectedMeasurements.map((e) => '(${e.size.toStringAsFixed(3)}, ${e.weight.toStringAsFixed(3)})').join(', ')}',
      );
      debugPrint(
        '  actual:   ${item.measurements.map((m) => '(${m.heightInMeters.toStringAsFixed(3)}, ${m.weightInKg.toStringAsFixed(3)})').join(', ')}',
      );
      if (item.entries.isNotEmpty && item.entries.first.candidates.isNotEmpty) {
        final top = item.entries.first.candidates.first;
        debugPrint(
          '  top1: ${top.petName} prob=${top.probability.toStringAsFixed(2)} '
          'height=${top.heightRangeLabel} weight=${top.weightRangeLabel}',
        );
      }
      debugPrint('');
    }

    final summary = {
      'generatedAt': report.generatedAt.toIso8601String(),
      'passedCount': report.passedCount,
      'totalCount': report.totalCount,
      'cases': report.cases
          .map(
            (item) => {
              'id': item.config.caseName,
              'group': item.config.sourceGroup,
              'passed': item.passed,
              'ocrLines': item.ocrDocument.lines
                  .map(
                    (line) => {
                      'text': line.text,
                      'left': line.bounds.left,
                      'top': line.bounds.top,
                      'width': line.bounds.width,
                      'height': line.bounds.height,
                    },
                  )
                  .toList(),
              'expected': item.config.expectedMeasurements
                  .map((m) => {'size': m.size, 'weight': m.weight})
                  .toList(),
              'actual': item.measurements
                  .map(
                    (m) => {
                      'size': m.heightInMeters,
                      'weight': m.weightInKg,
                    },
                  )
                  .toList(),
            },
          )
          .toList(),
    };

    final supportDir = await getApplicationSupportDirectory();
    final outputFile = File(
      '${supportDir.path}/ocr_benchmark_report.json',
    );
    await outputFile.writeAsString(
      const JsonEncoder.withIndent('  ').convert(summary),
      flush: true,
    );

    debugPrint('SUMMARY ${report.passedCount}/${report.totalCount} passed');
    debugPrint('REPORT ${outputFile.path}');
  } finally {
    container.dispose();
  }

  exit(0);
}
