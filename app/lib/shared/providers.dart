import 'dart:async';
import 'dart:typed_data';
import 'dart:ui';

import 'package:dio/dio.dart';
import 'package:egg_core/egg_core.dart';
import 'package:egg_data/egg_data.dart';
import 'package:egg_ocr/egg_ocr.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'desktop_window_controller.dart';

const appVersion = '0.1.0-dev.1';

final dioProvider = Provider<Dio>((ref) => Dio());

final datasetRepositoryProvider = Provider<AoetopDatasetRepository>((ref) {
  return AoetopDatasetRepository(dio: ref.watch(dioProvider));
});

final portraitRepositoryProvider = Provider<CachedPortraitRepository>((ref) {
  return CachedPortraitRepository(dio: ref.watch(dioProvider));
});

final ocrEngineProvider = Provider<OcrEngine>((ref) {
  return DefaultOcrEngine.create();
});

final analysisServiceProvider = Provider<AnalysisService>((ref) {
  return DefaultAnalysisService(
    ocrEngine: ref.watch(ocrEngineProvider),
    datasetRepository: ref.watch(datasetRepositoryProvider),
  );
});

final datasetSnapshotProvider = FutureProvider<DatasetSnapshot>((ref) async {
  final repository = ref.watch(datasetRepositoryProvider);
  final snapshot = await repository.load();
  if (await repository.isStale()) {
    unawaited(repository.refresh());
  }
  return snapshot;
});

final currentAnalysisResultProvider =
    StateProvider<AnalysisResult?>((ref) => null);

final desktopWindowControllerProvider =
    Provider<DesktopWindowController>((ref) {
  final controller = DesktopWindowController();
  ref.onDispose(() {
    controller.disposeController();
  });
  return controller;
});

final analysisControllerProvider = Provider<AnalysisController>((ref) {
  return AnalysisController(ref);
});

class AnalysisController {
  const AnalysisController(this.ref);

  static const _portraitPrefetchLimit = 6;
  static const _portraitPrefetchBatchSize = 2;

  final Ref ref;

  Future<AnalysisResult> analyzeBytes(
    Uint8List bytes, {
    required String label,
  }) async {
    final result = await ref.read(analysisServiceProvider).analyzeImage(
          bytes,
          sourceLabel: label,
        );
    await _preloadVisiblePortraits(result);
    ref.read(currentAnalysisResultProvider.notifier).state = result;
    return result;
  }

  Future<AnalysisResult> analyzeManualMeasurement({
    required double heightInMeters,
    required double weightInKg,
  }) async {
    final snapshot = await ref.read(datasetRepositoryProvider).load();
    final engine = MatchingEngine(snapshot.dataset);
    final measurement = Measurement(
      heightInMeters: heightInMeters,
      weightInKg: weightInKg,
      anchorText: '手动输入',
      anchor: Offset.zero,
      rawLines: const [],
    );
    final result = AnalysisResult(
      entries: [
        AnalysisEntry(
          measurement: measurement,
          candidates: engine.search(
            heightInCentimeters: heightInMeters * 100,
            weightInKg: weightInKg,
          ),
        ),
      ],
      ocrDocument: const OcrDocument(lines: []),
      sourceBytes: Uint8List(0),
      sourceLabel: '手动输入',
      analyzedAt: DateTime.now(),
    );
    await _preloadVisiblePortraits(result);
    ref.read(currentAnalysisResultProvider.notifier).state = result;
    return result;
  }

  Future<void> _preloadVisiblePortraits(AnalysisResult result) async {
    final portraitKeys = result.entries
        .expand((entry) => entry.candidates.take(2))
        .map((candidate) => candidate.portraitKey)
        .where((key) => key.trim().isNotEmpty)
        .take(_portraitPrefetchLimit)
        .toList(growable: false);
    if (portraitKeys.isEmpty) {
      return;
    }

    try {
      await ref.read(portraitRepositoryProvider).preloadPortraits(
            portraitKeys,
            batchSize: _portraitPrefetchBatchSize,
          );
    } catch (_) {
      // Portrait warmup is best-effort; analysis should still succeed.
    }
  }
}
