import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:egg_core/egg_core.dart';
import 'package:image/image.dart' as img;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

class RapidOcrDesktop {
  RapidOcrDesktop({
    String? pythonExecutable,
    String? workerScript,
    Directory? runtimeRoot,
    Map<String, String>? environment,
  })  : _pythonExecutable = pythonExecutable,
        _workerScript = workerScript ?? _defaultWorkerScript,
        _runtimeRootOverride = runtimeRoot,
        _environment = environment;

  final String? _pythonExecutable;
  final String _workerScript;
  final Directory? _runtimeRootOverride;
  final Map<String, String>? _environment;

  _WorkerState? _worker;
  Future<void> _queue = Future<void>.value();

  Future<OcrDocument> recognize(Uint8List imageBytes) {
    return _serialize(() async {
      final worker = await _ensureWorker();
      final dimensions = _decodeSize(imageBytes);
      final primary = await _sendRecognizeRequest(
        worker,
        imageBytes,
        mode: 'full',
      );
      if (!_shouldTryRecognitionOnly(
        dimensions,
        primary,
      )) {
        return primary;
      }

      final recognitionOnly = await _sendRecognizeRequest(
        worker,
        imageBytes,
        mode: 'rec_only',
      );
      return _scoreDocument(recognitionOnly) >= _scoreDocument(primary)
          ? recognitionOnly
          : primary;
    });
  }

  Future<OcrDocument> recognizeTextOnly(Uint8List imageBytes) {
    return _serialize(() async {
      final worker = await _ensureWorker();
      return _sendRecognizeRequest(worker, imageBytes, mode: 'rec_only');
    });
  }

  Future<List<OcrDocument>> recognizeTextOnlyBatch(
    List<Uint8List> imageBytesList,
  ) {
    return _serialize(() async {
      final worker = await _ensureWorker();
      return _sendRecognizeBatchRequest(
        worker,
        imageBytesList,
        mode: 'rec_only',
      );
    });
  }

  Future<void> dispose() {
    return _serialize(() async {
      final worker = _worker;
      _worker = null;
      if (worker == null) {
        return;
      }

      await worker.stdin.close();
      worker.stderrSubscription.cancel();
      worker.process.kill();
      await worker.process.exitCode.catchError((_) => -1);
      if (worker.ownsRuntimeRoot && await worker.runtimeRoot.exists()) {
        await worker.runtimeRoot.delete(recursive: true);
      }
    });
  }

  Future<T> _serialize<T>(Future<T> Function() action) {
    final completer = Completer<T>();
    _queue = _queue.then((_) async {
      try {
        completer.complete(await action());
      } catch (error, stackTrace) {
        completer.completeError(error, stackTrace);
      }
    });
    return completer.future;
  }

  Future<_WorkerState> _ensureWorker() async {
    final existing = _worker;
    if (existing != null) {
      return existing;
    }

    final runtimeRoot = await _resolveRuntimeRoot();
    await runtimeRoot.create(recursive: true);
    final scriptFile = File(p.join(runtimeRoot.path, 'rapidocr_runner.py'));
    await scriptFile.writeAsString(_workerScript, flush: true);

    final python = await _resolvePythonExecutable();
    final process = await Process.start(
      python,
      [scriptFile.path],
      environment: <String, String>{
        ...Platform.environment,
        'PYTHONUTF8': '1',
        'PYTHONIOENCODING': 'utf-8',
        ...?_environment,
      },
      workingDirectory: runtimeRoot.path,
      runInShell: Platform.isWindows,
    );

    final stderrBuffer = StringBuffer();
    final stderrSubscription = process.stderr
        .transform(utf8.decoder)
        .listen(stderrBuffer.write);
    final stdoutIterator = StreamIterator<String>(
      process.stdout.transform(utf8.decoder).transform(const LineSplitter()),
    );

    final worker = _WorkerState(
      process: process,
      stdin: process.stdin,
      stdout: stdoutIterator,
      stderrBuffer: stderrBuffer,
      stderrSubscription: stderrSubscription,
      runtimeRoot: runtimeRoot,
      ownsRuntimeRoot: _runtimeRootOverride == null,
    );

    try {
      if (!await stdoutIterator.moveNext()) {
        final exitCode = await process.exitCode;
        throw RapidOcrDesktopException(
          'RapidOCR worker failed to start (exit=$exitCode). '
          '${stderrBuffer.toString().trim()}',
        );
      }

      final readyPayload =
          jsonDecode(stdoutIterator.current) as Map<String, dynamic>;
      if (readyPayload['ready'] != true) {
        final message = (readyPayload['error'] as String?)?.trim();
        final traceback = (readyPayload['traceback'] as String?)?.trim();
        throw RapidOcrDesktopException(
          [
            'RapidOCR worker is unavailable.',
            if (message != null && message.isNotEmpty) message,
            if (traceback != null && traceback.isNotEmpty) traceback,
          ].join(' '),
        );
      }

      _worker = worker;
      return worker;
    } catch (_) {
      stderrSubscription.cancel();
      process.kill();
      if (_runtimeRootOverride == null && await runtimeRoot.exists()) {
        await runtimeRoot.delete(recursive: true);
      }
      rethrow;
    }
  }

  Future<Directory> _resolveRuntimeRoot() async {
    final override = _runtimeRootOverride;
    if (override != null) {
      return override;
    }

    final supportDirectory = await getApplicationSupportDirectory();
    return Directory(p.join(supportDirectory.path, 'rapidocr_desktop'));
  }

  Future<String> _resolvePythonExecutable() async {
    final overridePython = _pythonExecutable;
    if (overridePython != null && overridePython.trim().isNotEmpty) {
      return overridePython;
    }

    final candidates = <String>[
      Platform.environment['EGG_ANALYZE_PYTHON'] ?? '',
      ..._bundledPythonCandidates(),
      p.join('.venv', 'bin', 'python3'),
      p.join('.venv', 'bin', 'python'),
      p.join('.venv', 'Scripts', 'python.exe'),
      'python3',
      'python',
    ];

    for (final candidate in candidates) {
      final trimmed = candidate.trim();
      if (trimmed.isEmpty) {
        continue;
      }

      if (trimmed.contains(Platform.pathSeparator) ||
          trimmed.endsWith('.exe') ||
          trimmed.startsWith('.')) {
        if (await File(trimmed).exists()) {
          return trimmed;
        }
        continue;
      }

      final resolved = await _which(trimmed);
      if (resolved != null) {
        return resolved;
      }
    }

    throw const RapidOcrDesktopException(
      'RapidOCR requires Python. Configure EGG_ANALYZE_PYTHON, '
      'provide a bundled runtime, or install it in .venv.',
    );
  }

  List<String> _bundledPythonCandidates() {
    final executableDir = File(Platform.resolvedExecutable).parent.path;
    return <String>[
      p.join(executableDir, 'python', 'python.exe'),
      p.join(executableDir, 'python', 'bin', 'python3'),
      p.join(executableDir, 'python', 'bin', 'python'),
      p.join(executableDir, '..', 'Resources', 'python', 'bin', 'python3'),
      p.join(executableDir, '..', 'Resources', 'python', 'bin', 'python'),
    ];
  }

  Future<String?> _which(String executable) async {
    final tool = Platform.isWindows ? 'where' : 'which';
    try {
      final result = await Process.run(tool, [executable], runInShell: true);
      if (result.exitCode != 0) {
        return null;
      }
      final output = (result.stdout as String).trim();
      if (output.isEmpty) {
        return null;
      }
      return output.split(RegExp(r'\r?\n')).first.trim();
    } catch (_) {
      return null;
    }
  }

  Future<OcrDocument> _sendRecognizeRequest(
    _WorkerState worker,
    Uint8List imageBytes,
    {
    required String mode,
  }) async {
    final payload = jsonEncode(<String, Object?>{
      'image_base64': base64Encode(imageBytes),
      'mode': mode,
    });
    worker.stdin.writeln(payload);
    await worker.stdin.flush();

    if (!await worker.stdout.moveNext()) {
      final exitCode = await worker.process.exitCode;
      final stderr = worker.stderrBuffer.toString().trim();
      _worker = null;
      throw RapidOcrDesktopException(
        'RapidOCR worker exited unexpectedly (exit=$exitCode). $stderr',
      );
    }

    final response =
        jsonDecode(worker.stdout.current) as Map<String, dynamic>;
    if (response['ok'] != true) {
      final error = (response['error'] as String?)?.trim();
      final traceback = (response['traceback'] as String?)?.trim();
      throw RapidOcrDesktopException(
        [
          'RapidOCR failed to recognize the image.',
          if (error != null && error.isNotEmpty) error,
          if (traceback != null && traceback.isNotEmpty) traceback,
          worker.stderrBuffer.toString().trim(),
        ].where((item) => item.isNotEmpty).join(' '),
      );
    }

    final rawLines =
        (response['lines'] as List<dynamic>? ?? const <dynamic>[]);
    return OcrDocument(
      lines: rawLines
          .whereType<Map<String, dynamic>>()
          .map(_toOcrLine)
          .where((line) => line.text.trim().isNotEmpty)
          .toList(growable: false),
    );
  }

  Future<List<OcrDocument>> _sendRecognizeBatchRequest(
    _WorkerState worker,
    List<Uint8List> imageBytesList, {
    required String mode,
  }) async {
    final payload = jsonEncode(<String, Object?>{
      'images_base64': imageBytesList.map(base64Encode).toList(growable: false),
      'mode': mode,
    });
    worker.stdin.writeln(payload);
    await worker.stdin.flush();

    if (!await worker.stdout.moveNext()) {
      final exitCode = await worker.process.exitCode;
      final stderr = worker.stderrBuffer.toString().trim();
      _worker = null;
      throw RapidOcrDesktopException(
        'RapidOCR worker exited unexpectedly (exit=$exitCode). $stderr',
      );
    }

    final response =
        jsonDecode(worker.stdout.current) as Map<String, dynamic>;
    if (response['ok'] != true) {
      final error = (response['error'] as String?)?.trim();
      final traceback = (response['traceback'] as String?)?.trim();
      throw RapidOcrDesktopException(
        [
          'RapidOCR failed to recognize the image batch.',
          if (error != null && error.isNotEmpty) error,
          if (traceback != null && traceback.isNotEmpty) traceback,
          worker.stderrBuffer.toString().trim(),
        ].where((item) => item.isNotEmpty).join(' '),
      );
    }

    final rawDocuments =
        (response['documents'] as List<dynamic>? ?? const <dynamic>[]);
    return rawDocuments
        .whereType<Map<String, dynamic>>()
        .map(
          (document) => OcrDocument(
            lines: (document['lines'] as List<dynamic>? ?? const <dynamic>[])
                .whereType<Map<String, dynamic>>()
                .map(_toOcrLine)
                .where((line) => line.text.trim().isNotEmpty)
                .toList(growable: false),
          ),
        )
        .toList(growable: false);
  }

  OcrLine _toOcrLine(Map<String, dynamic> json) {
    double toDouble(Object? value) {
      if (value is num) {
        return value.toDouble();
      }
      return double.tryParse('$value') ?? 0;
    }

    return OcrLine(
      text: (json['text'] as String? ?? '').trim(),
      bounds: Rect.fromLTWH(
        toDouble(json['left']),
        toDouble(json['top']),
        toDouble(json['width']),
        toDouble(json['height']),
      ),
    );
  }

  _ImageDimensions? _decodeSize(Uint8List imageBytes) {
    final decoded = img.decodeImage(imageBytes);
    if (decoded == null) {
      return null;
    }
    return _ImageDimensions(width: decoded.width, height: decoded.height);
  }

  bool _shouldTryRecognitionOnly(
    _ImageDimensions? dimensions,
    OcrDocument primary,
  ) {
    if (dimensions == null) {
      return false;
    }

    final width = dimensions.width;
    final height = dimensions.height;
    final longestEdge = width > height ? width : height;
    final shortestEdge = width < height ? width : height;
    final aspectRatio = height == 0 ? 0 : width / height;
    final looksLikeRescueCrop =
        longestEdge <= 320 || (shortestEdge <= 96 && aspectRatio >= 1.8);
    if (!looksLikeRescueCrop) {
      return false;
    }

    final score = _scoreDocument(primary);
    final decimalLikeLines = primary.lines
        .where((line) => RegExp(r'\d\.\d').hasMatch(line.text))
        .length;
    return primary.lines.isEmpty || decimalLikeLines == 0 || score <= 4;
  }

  int _scoreDocument(OcrDocument document) {
    var score = 0;
    for (final line in document.lines) {
      if (RegExp(r'\d').hasMatch(line.text)) {
        score += 1;
      }
      if (RegExp(r'\d\.\d').hasMatch(line.text)) {
        score += 3;
      }
      if (line.text.length >= 4) {
        score += 1;
      }
    }
    return score;
  }
}

class RapidOcrDesktopException implements Exception {
  const RapidOcrDesktopException(this.message);

  final String message;

  @override
  String toString() => message;
}

class _WorkerState {
  _WorkerState({
    required this.process,
    required this.stdin,
    required this.stdout,
    required this.stderrBuffer,
    required this.stderrSubscription,
    required this.runtimeRoot,
    required this.ownsRuntimeRoot,
  });

  final Process process;
  final IOSink stdin;
  final StreamIterator<String> stdout;
  final StringBuffer stderrBuffer;
  final StreamSubscription<String> stderrSubscription;
  final Directory runtimeRoot;
  final bool ownsRuntimeRoot;
}

class _ImageDimensions {
  const _ImageDimensions({
    required this.width,
    required this.height,
  });

  final int width;
  final int height;
}

const String _defaultWorkerScript = r'''
import base64
import json
import sys
import traceback


def bounds(box):
    xs = [int(point[0]) for point in box]
    ys = [int(point[1]) for point in box]
    left = min(xs)
    top = min(ys)
    right = max(xs)
    bottom = max(ys)
    return left, top, right - left, bottom - top


def emit(payload):
    print(json.dumps(payload), flush=True)


def to_lines(result, mode):
    lines = []
    if mode == "rec_only":
        for item in result or []:
            lines.append(
                {
                    "text": item[0],
                    "left": 0,
                    "top": 0,
                    "width": 0,
                    "height": 0,
                }
            )
        return lines

    for item in result or []:
        box, text, _score = item
        left, top, width, height = bounds(box)
        lines.append(
            {
                "text": text,
                "left": left,
                "top": top,
                "width": width,
                "height": height,
            }
        )
    return lines


def main():
    if hasattr(sys.stdin, "reconfigure"):
        sys.stdin.reconfigure(encoding="utf-8")
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")

    try:
        from rapidocr_onnxruntime import RapidOCR

        engine = RapidOCR(
            det_limit_side_len=960,
            det_box_thresh=0.42,
            max_side_len=2560,
            width_height_ratio=12,
        )
    except Exception as exc:
        emit(
            {
                "ready": False,
                "error": str(exc),
                "traceback": traceback.format_exc(limit=1),
            }
        )
        return 1

    emit({"ready": True})

    for raw in sys.stdin:
        raw = raw.strip()
        if not raw:
            continue
        try:
            payload = json.loads(raw)
            mode = payload.get("mode", "full")
            if "images_base64" in payload:
                documents = []
                for encoded in payload["images_base64"]:
                    image_bytes = base64.b64decode(encoded)
                    if mode == "rec_only":
                        result, _ = engine(
                            image_bytes,
                            use_det=False,
                            use_cls=False,
                            use_rec=True,
                        )
                    else:
                        result, _ = engine(image_bytes)
                    documents.append({"lines": to_lines(result, mode)})
                emit({"ok": True, "documents": documents})
                continue

            image_bytes = base64.b64decode(payload["image_base64"])
            if mode == "rec_only":
                result, _ = engine(
                    image_bytes,
                    use_det=False,
                    use_cls=False,
                    use_rec=True,
                )
            else:
                result, _ = engine(image_bytes)

            emit({"ok": True, "lines": to_lines(result, mode)})
        except Exception as exc:
            emit(
                {
                    "ok": False,
                    "error": str(exc),
                    "traceback": traceback.format_exc(limit=1),
                }
            )


if __name__ == "__main__":
    raise SystemExit(main())
''';
