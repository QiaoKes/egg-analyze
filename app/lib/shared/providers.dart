import 'dart:async';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:egg_core/egg_core.dart';
import 'package:egg_data/egg_data.dart';
import 'package:egg_ocr/egg_ocr.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'recent_analysis_store.dart';

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

final recentAnalysisStoreProvider = Provider<RecentAnalysisStore>(
  (ref) => RecentAnalysisStore(),
);

final recentRecordsProvider =
    AsyncNotifierProvider<RecentRecordsController, List<RecentAnalysisRecord>>(
  RecentRecordsController.new,
);

final analysisControllerProvider = Provider<AnalysisController>((ref) {
  return AnalysisController(ref);
});

class RecentRecordsController
    extends AsyncNotifier<List<RecentAnalysisRecord>> {
  @override
  Future<List<RecentAnalysisRecord>> build() async {
    return ref.read(recentAnalysisStoreProvider).load();
  }

  Future<void> addRecord(RecentAnalysisRecord record) async {
    await ref.read(recentAnalysisStoreProvider).add(record);
    state = AsyncData(await ref.read(recentAnalysisStoreProvider).load());
  }

  Future<void> clear() async {
    await ref.read(recentAnalysisStoreProvider).clear();
    state = const AsyncData(<RecentAnalysisRecord>[]);
  }
}

class AnalysisController {
  const AnalysisController(this.ref);

  final Ref ref;

  Future<AnalysisResult> analyzeBytes(
    Uint8List bytes, {
    required String label,
  }) async {
    final result = await ref.read(analysisServiceProvider).analyzeImage(
          bytes,
          sourceLabel: label,
        );
    ref.read(currentAnalysisResultProvider.notifier).state = result;
    await ref.read(recentRecordsProvider.notifier).addRecord(
          RecentAnalysisRecord(
            id: '${DateTime.now().microsecondsSinceEpoch}',
            label: label,
            createdAt: DateTime.now(),
          ),
        );
    return result;
  }
}
