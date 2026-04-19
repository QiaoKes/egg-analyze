import 'dart:typed_data';

import 'models.dart';

abstract class OcrEngine {
  Future<OcrDocument> recognize(Uint8List imageBytes);
}

abstract class TextOnlyOcrEngine implements OcrEngine {
  Future<OcrDocument> recognizeTextOnly(Uint8List imageBytes);
}

abstract class BatchTextOnlyOcrEngine implements TextOnlyOcrEngine {
  Future<List<OcrDocument>> recognizeTextOnlyBatch(List<Uint8List> imageBytesList);
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
