import 'dart:typed_data';

import 'models.dart';

abstract class OcrEngine {
  Future<OcrDocument> recognize(Uint8List imageBytes);
}

abstract class DatasetRepository {
  Future<DatasetSnapshot> load();
  Future<DatasetSnapshot> refresh();
}

abstract class AnalysisService {
  Future<AnalysisResult> analyzeImage(
    Uint8List imageBytes, {
    String sourceLabel = '导入图片',
  });
}
