import 'dart:typed_data';

import 'contracts.dart';
import 'matching_engine.dart';
import 'measurement_extractor.dart';
import 'models.dart';

class DefaultAnalysisService implements AnalysisService {
  DefaultAnalysisService({
    required OcrEngine ocrEngine,
    required DatasetRepository datasetRepository,
    MeasurementExtractor extractor = const MeasurementExtractor(),
  })  : _ocrEngine = ocrEngine,
        _datasetRepository = datasetRepository,
        _extractor = extractor;

  final OcrEngine _ocrEngine;
  final DatasetRepository _datasetRepository;
  final MeasurementExtractor _extractor;

  @override
  Future<AnalysisResult> analyzeImage(
    Uint8List imageBytes, {
    String sourceLabel = '导入图片',
  }) async {
    final snapshot = await _datasetRepository.load();
    final engine = MatchingEngine(snapshot.dataset);
    final ocrDocument = await _ocrEngine.recognize(imageBytes);
    final recognizeCrop = _ocrEngine is TextOnlyOcrEngine
        ? (_ocrEngine as TextOnlyOcrEngine).recognizeTextOnly
        : _ocrEngine.recognize;
    final recognizeCropBatch = _ocrEngine is BatchTextOnlyOcrEngine
        ? (_ocrEngine as BatchTextOnlyOcrEngine).recognizeTextOnlyBatch
        : null;
    final measurements = await _extractor.extractBestMeasurements(
      lines: ocrDocument.lines,
      priors: engine.measurementPriors,
      sourceBytes: imageBytes,
      recognizeCrop: recognizeCrop,
      recognizeCropBatch: recognizeCropBatch,
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

    return AnalysisResult(
      entries: entries,
      ocrDocument: ocrDocument,
      sourceBytes: imageBytes,
      sourceLabel: sourceLabel,
      analyzedAt: DateTime.now(),
    );
  }
}
