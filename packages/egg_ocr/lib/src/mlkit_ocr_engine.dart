import 'dart:io';
import 'dart:typed_data';

import 'package:egg_core/egg_core.dart';
import 'package:google_mlkit_text_recognition/google_mlkit_text_recognition.dart';
import 'package:path_provider/path_provider.dart';

class MlKitOcrEngine implements OcrEngine {
  MlKitOcrEngine({
    TextRecognizer? recognizer,
  }) : _recognizer =
            recognizer ?? TextRecognizer(script: TextRecognitionScript.latin);

  final TextRecognizer _recognizer;

  @override
  Future<OcrDocument> recognize(Uint8List imageBytes) async {
    final tempDir = await getTemporaryDirectory();
    final file = File(
      '${tempDir.path}/egg-analyze-ocr-${DateTime.now().microsecondsSinceEpoch}.png',
    );
    await file.writeAsBytes(imageBytes, flush: true);
    try {
      final inputImage = InputImage.fromFilePath(file.path);
      final recognizedText = await _recognizer.processImage(inputImage);
      final lines = <OcrLine>[];
      for (final block in recognizedText.blocks) {
        for (final line in block.lines) {
          lines.add(
            OcrLine(
              text: line.text,
              bounds: line.boundingBox,
            ),
          );
        }
      }
      return OcrDocument(lines: lines);
    } finally {
      if (await file.exists()) {
        await file.delete();
      }
    }
  }
}
