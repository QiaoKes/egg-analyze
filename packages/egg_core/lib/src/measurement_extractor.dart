import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui';

import 'package:image/image.dart' as img;

import 'models.dart';

class MeasurementExtractor {
  const MeasurementExtractor();

  Future<List<Measurement>> extractBestMeasurements({
    required List<OcrLine> lines,
    required MeasurementPriors priors,
    Uint8List? sourceBytes,
    Future<OcrDocument> Function(Uint8List imageBytes)? recognizeCrop,
  }) async {
    final base = extractMeasurements(lines, priors);
    if (sourceBytes == null || recognizeCrop == null) {
      return base;
    }

    final decoded = img.decodeImage(sourceBytes);
    if (decoded == null) {
      return base;
    }

    final rescued = await _rescueMeasurements(
      sourceImage: decoded,
      lines: lines,
      existing: base,
      priors: priors,
      recognizeCrop: recognizeCrop,
    );

    final combined = _dedupeMeasurements([
      ...base,
      ...rescued,
    ]);
    return _refineWeightLineMeasurements(
      measurements: combined,
      sourceImage: decoded,
      weightRange: priors.weight,
      recognizeCrop: recognizeCrop,
    );
  }

  List<Measurement> extractMeasurements(
    List<OcrLine> lines,
    MeasurementPriors priors,
  ) {
    final sortedLines = [...lines]..sort((left, right) {
        if ((left.y - right.y).abs() <= _rowTolerance(left, right)) {
          return left.x.compareTo(right.x);
        }
        return left.y.compareTo(right.y);
      });

    final values = <_ExtractionCandidate>[];
    for (var index = 0; index < sortedLines.length; index++) {
      final line = sortedLines[index];
      final parsed = _parseNumber(line.text);
      if (parsed == null ||
          _isIgnoredNumberLine(line.text, parsed.normalized)) {
        continue;
      }

      final unionRange = _unionRange(priors.diameter, priors.weight);
      if (!_inRange(parsed.value, _expandRange(unionRange, 0.35, 1))) {
        continue;
      }

      values.add(
        _ExtractionCandidate(
          value: parsed.value,
          normalized: parsed.normalized,
          line: line,
          order: index,
        ),
      );
    }

    final pairs = <_CandidatePair>[];
    for (var i = 0; i < values.length; i++) {
      for (var j = i + 1; j < values.length; j++) {
        final score = _scorePair(values[i], values[j], priors);
        if (score == null) {
          continue;
        }
        pairs.add(
            _CandidatePair(left: values[i], right: values[j], score: score));
      }
    }

    pairs.sort((left, right) {
      if ((left.score - right.score).abs() < 0.001) {
        if (left.left.order == right.left.order) {
          return left.right.order.compareTo(right.right.order);
        }
        return left.left.order.compareTo(right.left.order);
      }
      return right.score.compareTo(left.score);
    });

    final used = <int>{};
    final results = <Measurement>[];
    for (final pair in pairs) {
      if (used.contains(pair.left.order) || used.contains(pair.right.order)) {
        continue;
      }
      used.add(pair.left.order);
      used.add(pair.right.order);
      results.add(
        Measurement(
          heightInMeters: pair.left.value,
          weightInKg: pair.right.value,
          anchorText: pair.left.line.text,
          anchor:
              Offset(pair.left.line.x.toDouble(), pair.left.line.y.toDouble()),
          rawLines: lines,
          weightText: pair.right.line.text,
          weightBounds: Rect.fromLTWH(
            pair.right.line.x.toDouble(),
            pair.right.line.y.toDouble(),
            pair.right.line.width.toDouble(),
            pair.right.line.height.toDouble(),
          ),
        ),
      );
    }

    results.sort((left, right) {
      if ((left.anchor.dy - right.anchor.dy).abs() <= 12) {
        return left.anchor.dx.compareTo(right.anchor.dx);
      }
      return left.anchor.dy.compareTo(right.anchor.dy);
    });

    return _dedupeMeasurements(results);
  }

  _ParsedNumber? _parseNumber(String input) {
    final normalized = _normalizeNumericText(input);
    final candidates = <String>{};
    for (final match in RegExp(r'(?:\d+\.\d+|\.\d+|\d+)').allMatches(normalized)) {
      candidates.add(match.group(0)!);
    }
    for (final match in RegExp(r'0\.\d+').allMatches(normalized)) {
      candidates.add(match.group(0)!);
    }
    if (candidates.isEmpty) {
      return null;
    }

    var valueText = candidates.first;
    final prefersEmbeddedDecimal = candidates.any((item) => item.startsWith('0.')) &&
        candidates.any((item) => (double.tryParse(item) ?? 0) > 10);
    if (prefersEmbeddedDecimal) {
      valueText = candidates
          .where((item) => item.startsWith('0.'))
          .reduce((left, right) => left.length >= right.length ? left : right);
    }
    if (valueText.startsWith('.')) {
      valueText = '0$valueText';
    }
    final value = double.tryParse(valueText);
    if (value == null) {
      return null;
    }
    return _ParsedNumber(value: value, normalized: valueText);
  }

  String _normalizeNumericText(String input) {
    var normalized = input
        .replaceAll(' ', '')
        .replaceAll('　', '')
        .replaceAll('．', '.')
        .replaceAll('。', '.')
        .replaceAll('·', '.')
        .replaceAll(',', '.');

    final runes = normalized.runes.toList();
    for (var i = 0; i < runes.length; i++) {
      final mapped = _mapDigitLikeRune(runes[i]);
      if (mapped == null) {
        continue;
      }
      if (!_isNumericContextRune(runes, i)) {
        continue;
      }
      runes[i] = mapped;
    }
    return String.fromCharCodes(runes);
  }

  int? _mapDigitLikeRune(int rune) {
    switch (String.fromCharCode(rune)) {
      case 'O':
      case 'o':
      case 'Q':
      case 'D':
      case '□':
      case '口':
      case '〇':
        return '0'.codeUnitAt(0);
      case 'I':
      case 'i':
      case 'l':
      case '|':
      case '!':
        return '1'.codeUnitAt(0);
      case 'Z':
      case 'z':
        return '2'.codeUnitAt(0);
      case 'S':
      case 's':
      case 'E':
      case 'e':
      case 'c':
      case 'C':
      case 'г':
      case 'Г':
        return '5'.codeUnitAt(0);
      case 'B':
      case 'b':
        return '3'.codeUnitAt(0);
      default:
        return null;
    }
  }

  bool _isNumericContextRune(List<int> values, int index) {
    final leftNumeric = index > 0 && _isDigitOrDot(values[index - 1]);
    final rightNumeric =
        index + 1 < values.length && _isDigitOrDot(values[index + 1]);
    return leftNumeric || rightNumeric;
  }

  bool _isDigitOrDot(int rune) {
    final value = String.fromCharCode(rune);
    return RegExp(r'[0-9.]').hasMatch(value);
  }

  bool _isIgnoredNumberLine(String raw, String normalized) {
    final trimmed = normalized.trim();
    if (trimmed.isEmpty) {
      return true;
    }
    if (raw.contains('%')) {
      return true;
    }
    if (RegExp(r'^\d{4}[-/.]\d{1,2}[-/.]\d{1,2}$').hasMatch(trimmed)) {
      return true;
    }
    return !trimmed.contains('.');
  }

  double? _scorePair(
    _ExtractionCandidate left,
    _ExtractionCandidate right,
    MeasurementPriors priors,
  ) {
    final dy = (right.line.y - left.line.y).abs();
    final dx = right.line.x - left.line.x;
    final sameRow = dy <= _rowTolerance(left.line, right.line);
    final sameColumn = (right.line.x - left.line.x).abs() <=
        math.max(left.line.width, right.line.width) * 2;

    switch ((sameRow, sameColumn)) {
      case (true, _):
        if (dx <= 0 || dx > 520) {
          return null;
        }
      case (false, true):
        if (dy <= 0 || dy > 220) {
          return null;
        }
      default:
        return null;
    }

    var score = 0.0;
    score += _plausibilityScore(left.value, priors.diameter, 0.18);
    score += _plausibilityScore(right.value, priors.weight, 0.18);
    if (score <= 40) {
      return null;
    }

    if (sameRow) {
      score += 140;
      score += 60 * _closenessScore(dx.toDouble(), 520);
      score += 30 *
          _closenessScore(
            dy.toDouble(),
            math.max(16, _rowTolerance(left.line, right.line)).toDouble(),
          );
    } else {
      score += 90;
      score += 50 * _closenessScore(dy.toDouble(), 220);
      score += 20 *
          _closenessScore(
            (right.line.x - left.line.x).abs().toDouble(),
            math
                .max(80, math.max(left.line.width, right.line.width) * 2)
                .toDouble(),
          );
    }

    final gap = right.order - left.order - 1;
    if (gap > 0) {
      score -= gap * 18;
    }
    if ('.'.allMatches(left.normalized).length == 1) {
      score += 8;
    }
    if ('.'.allMatches(right.normalized).length == 1) {
      score += 8;
    }
    return score;
  }

  double _plausibilityScore(double value, Range expected, double margin) {
    final expanded = _expandRange(expected, margin, 0.02);
    if (!_inRange(value, expanded)) {
      return -120;
    }
    if (_inRange(value, expected)) {
      return 80 +
          20 *
              _closenessScore(
                _distanceToCenter(value, expected),
                _rangeHalfWidth(expected) + 0.01,
              );
    }
    return 45 +
        15 *
            _closenessScore(
                _distanceToRange(value, expected), expanded.max - expanded.min);
  }

  int _rowTolerance(OcrLine left, OcrLine right) {
    final height = math.max(left.height, right.height);
    return math.max(12, height ~/ 2);
  }

  Range _unionRange(Range left, Range right) => Range(
        min: math.min(left.min, right.min),
        max: math.max(left.max, right.max),
      );

  Range _expandRange(Range range, double ratio, double floor) {
    final span = range.max - range.min;
    final margin = math.max(span * ratio, floor);
    return Range(
      min: math.max(0, range.min - margin),
      max: range.max + margin,
    );
  }

  bool _inRange(double value, Range range) =>
      value >= range.min && value <= range.max;

  double _distanceToCenter(double value, Range range) {
    final center = (range.min + range.max) / 2;
    return (value - center).abs();
  }

  double _rangeHalfWidth(Range range) {
    return math.max((range.max - range.min) / 2, 0.001);
  }

  double _distanceToRange(double value, Range range) {
    if (value < range.min) {
      return range.min - value;
    }
    if (value > range.max) {
      return value - range.max;
    }
    return 0;
  }

  double _closenessScore(double distance, double scale) {
    if (scale <= 0) {
      return 0;
    }
    final score = 1 - distance / scale;
    return score < 0 ? 0 : score;
  }

  List<Measurement> _dedupe(List<Measurement> values) {
    return _dedupeMeasurements(values);
  }

  List<Measurement> _dedupeMeasurements(List<Measurement> values) {
    final seen = <String>{};
    final filtered = <Measurement>[];
    for (final value in values) {
      final key =
          '${value.heightInMeters.toStringAsFixed(3)}-${value.weightInKg.toStringAsFixed(3)}';
      if (seen.add(key)) {
        filtered.add(value);
      }
    }
    return filtered;
  }

  Future<List<Measurement>> _rescueMeasurements({
    required img.Image sourceImage,
    required List<OcrLine> lines,
    required List<Measurement> existing,
    required MeasurementPriors priors,
    required Future<OcrDocument> Function(Uint8List imageBytes) recognizeCrop,
  }) async {
    final matched = <String>{
      for (final item in existing)
        '${item.anchor.dx.round()}:${item.anchor.dy.round()}',
    };
    final consumedWeightLines = <String>{
      for (final item in existing)
        if (item.weightBounds != null)
          '${item.weightBounds!.left.round()}:${item.weightBounds!.top.round()}',
    };

    final sortedLines = [...lines]..sort((left, right) {
        if ((left.y - right.y).abs() <= _rowTolerance(left, right)) {
          return left.x.compareTo(right.x);
        }
        return left.y.compareTo(right.y);
      });

    final rescued = <Measurement>[];
    for (final line in sortedLines) {
      final anchorKey = '${line.x}:${line.y}';
      if (matched.contains(anchorKey)) {
        continue;
      }
      if (consumedWeightLines.contains(anchorKey)) {
        continue;
      }

      final parsed = _parseNumber(line.text);
      if (parsed == null ||
          _isIgnoredNumberLine(line.text, parsed.normalized) ||
          !_inRange(parsed.value, _expandRange(priors.diameter, 0.18, 0.02))) {
        continue;
      }

      final weight = await _rescueWeightBelowAnchor(
        sourceImage: sourceImage,
        anchor: line,
        lines: sortedLines,
        weightRange: priors.weight,
        recognizeCrop: recognizeCrop,
      );
      if (weight == null) {
        continue;
      }
      if ((weight - parsed.value).abs() <= 0.05) {
        continue;
      }

      rescued.add(
        Measurement(
          heightInMeters: parsed.value,
          weightInKg: weight,
          anchorText: line.text,
          anchor: Offset(line.x.toDouble(), line.y.toDouble()),
          rawLines: lines,
        ),
      );
      matched.add(anchorKey);
    }

    return rescued;
  }

  Future<List<Measurement>> _refineWeightLineMeasurements({
    required List<Measurement> measurements,
    required img.Image sourceImage,
    required Range weightRange,
    required Future<OcrDocument> Function(Uint8List imageBytes) recognizeCrop,
  }) async {
    final refined = <Measurement>[];
    for (final item in measurements) {
      final weightBounds = item.weightBounds;
      if (weightBounds == null) {
        refined.add(item);
        continue;
      }

      final refinedWeight = await _refineWeightFromExactLineCrop(
        sourceImage: sourceImage,
        weightBounds: weightBounds,
        weightRange: weightRange,
        recognizeCrop: recognizeCrop,
      );
      if (refinedWeight == null ||
          (refinedWeight.value - item.weightInKg).abs() > 0.02) {
        refined.add(item);
        continue;
      }

      refined.add(
        Measurement(
          heightInMeters: item.heightInMeters,
          weightInKg: refinedWeight.value,
          anchorText: item.anchorText,
          anchor: item.anchor,
          rawLines: item.rawLines,
          weightText: item.weightText,
          weightBounds: item.weightBounds,
        ),
      );
    }
    return _dedupeMeasurements(refined);
  }

  Future<_RescuedWeight?> _refineWeightFromExactLineCrop({
    required img.Image sourceImage,
    required Rect weightBounds,
    required Range weightRange,
    required Future<OcrDocument> Function(Uint8List imageBytes) recognizeCrop,
  }) async {
    final candidates = <String, List<_RescuedWeight>>{};
    for (final padding in const [(16, 6), (16, 10), (20, 10)]) {
      final crop = _cropRect(
        sourceImage,
        weightBounds.left.round() - padding.$1,
        weightBounds.top.round() - padding.$2,
        weightBounds.right.round() + padding.$1,
        weightBounds.bottom.round() + padding.$2,
      );
      if (crop == null) {
        continue;
      }

      final variants = <img.Image>[
        crop,
        img.grayscale(crop),
      ];

      for (final variant in variants) {
        final bytes = Uint8List.fromList(img.encodePng(variant));
        final document = await recognizeCrop(bytes);
        final parsed = _parseBestRescuedWeight(document.lines, weightRange);
        if (parsed == null) {
          continue;
        }
        final key = parsed.value.toStringAsFixed(3);
        candidates.putIfAbsent(key, () => <_RescuedWeight>[]).add(parsed);
      }
    }

    if (candidates.isEmpty) {
      return null;
    }

    _RescuedWeight? best;
    var bestCount = -1;
    for (final group in candidates.values) {
      group.sort((left, right) => right.score.compareTo(left.score));
      final leader = group.first;
      if (group.length > bestCount) {
        best = leader;
        bestCount = group.length;
        continue;
      }
      if (group.length == bestCount &&
          best != null &&
          leader.score > best.score) {
        best = leader;
      }
    }

    return best;
  }

  Future<double?> _rescueWeightBelowAnchor({
    required img.Image sourceImage,
    required OcrLine anchor,
    required List<OcrLine> lines,
    required Range weightRange,
    required Future<OcrDocument> Function(Uint8List imageBytes) recognizeCrop,
  }) async {
    final result = await _rescueWeightBelowAnchorDetailed(
      sourceImage: sourceImage,
      anchor: anchor,
      lines: lines,
      weightRange: weightRange,
      recognizeCrop: recognizeCrop,
    );
    return result?.value;
  }

  Future<_RescuedWeight?> _rescueWeightBelowAnchorDetailed({
    required img.Image sourceImage,
    required OcrLine anchor,
    required List<OcrLine> lines,
    required Range weightRange,
    required Future<OcrDocument> Function(Uint8List imageBytes) recognizeCrop,
  }) async {
    var bestScore = double.negativeInfinity;
    _RescuedWeight? best;

    for (final crop in _buildRescueCrops(sourceImage, anchor, lines)) {
      for (final prepared in _prepareRescueImages(crop)) {
        final bytes = Uint8List.fromList(img.encodePng(prepared));
        final document = await recognizeCrop(bytes);
        final parsed = _parseBestRescuedWeight(document.lines, weightRange);
        if (parsed == null) {
          continue;
        }
        if (parsed.score > bestScore) {
          bestScore = parsed.score;
          best = parsed;
        }
      }
    }

    return best;
  }

  List<img.Image> _buildRescueCrops(
    img.Image source,
    OcrLine anchor,
    List<OcrLine> lines,
  ) {
    final crops = <img.Image>[];

    final target = _findRescueTarget(anchor, lines);
    if (target != null) {
      final crop = _cropLineWithPadding(source, target, 18, 12);
      if (crop != null) {
        crops.add(crop);
      }
    }

    final blindA =
        _cropWeightBelowAnchor(source, anchor, 0.10, 1.20, 1.30, 1.55);
    if (blindA != null) {
      crops.add(blindA);
    }
    final blindB =
        _cropWeightBelowAnchor(source, anchor, 0.00, 1.12, 1.18, 1.70);
    if (blindB != null) {
      crops.add(blindB);
    }

    return crops;
  }

  img.Image? _cropWeightBelowAnchor(
    img.Image source,
    OcrLine anchor,
    double leftPadRatio,
    double topOffsetRatio,
    double widthRatio,
    double heightRatio,
  ) {
    final left = anchor.x - (anchor.width * leftPadRatio).round();
    final top = anchor.y + (anchor.height * topOffsetRatio).round();
    final right = anchor.x + (anchor.width * widthRatio).round();
    final bottom = top + (anchor.height * heightRatio).round();
    return _cropRect(source, left, top, right, bottom);
  }

  List<img.Image> _prepareRescueImages(img.Image source) {
    final images = <img.Image>[source];
    final height = source.height;
    if (height > 0 && height < 28) {
      images.add(img.copyResize(source,
          width: source.width * 3, height: source.height * 3));
    } else if (height >= 28 && height < 64) {
      images.add(img.copyResize(source,
          width: source.width * 2, height: source.height * 2));
    }
    return images;
  }

  _RescuedWeight? _parseBestRescuedWeight(
      List<OcrLine> lines, Range weightRange) {
    var bestScore = -1.0;
    double? bestValue;
    final expanded = _expandRange(weightRange, 0.25, 0.03);
    for (final line in lines) {
      final parsed = _parseNumber(line.text);
      if (parsed == null ||
          _isIgnoredNumberLine(line.text, parsed.normalized) ||
          !_inRange(parsed.value, expanded)) {
        continue;
      }

      final score = _plausibilityScore(parsed.value, weightRange, 0.12) +
          _rescuedPrecisionScore(parsed.normalized);
      if (score > bestScore) {
        bestScore = score;
        bestValue = parsed.value;
      }
    }

    if (bestValue == null || bestScore <= 0) {
      return null;
    }
    return _RescuedWeight(value: bestValue, score: bestScore);
  }

  double _rescuedPrecisionScore(String normalized) {
    final decimalIndex = normalized.indexOf('.');
    if (decimalIndex == -1 || decimalIndex == normalized.length - 1) {
      return 0;
    }
    final tail = normalized.substring(decimalIndex + 1);
    final digits = tail.runes.takeWhile((r) => r >= 48 && r <= 57).length;
    var score = math.min(digits, 3) * 3.0;
    if (digits >= 3) {
      score += 1;
    }
    return score;
  }

  OcrLine? _findRescueTarget(OcrLine anchor, List<OcrLine> lines) {
    var bestScore = double.negativeInfinity;
    OcrLine? best;
    for (final candidate in lines) {
      if (candidate.x == anchor.x &&
          candidate.y == anchor.y &&
          candidate.width == anchor.width &&
          candidate.height == anchor.height) {
        continue;
      }

      final dy = candidate.y - anchor.y;
      final dx = candidate.x - anchor.x;
      final sameColumn = (dx).abs() <= math.max(anchor.width, candidate.width);
      final sameRow = dy.abs() <= _rowTolerance(anchor, candidate);

      double? score;
      if (sameColumn && dy > 0 && dy <= math.max(180, anchor.height * 4)) {
        score = 220 - dy - dx.abs() * 0.8;
      } else if (sameRow && dx > 0 && dx <= math.max(260, anchor.width * 5)) {
        score = 180 - dx - dy.abs() * 1.2;
      }

      if (score != null && score > bestScore) {
        bestScore = score;
        best = candidate;
      }
    }
    return best;
  }

  img.Image? _cropLineWithPadding(
      img.Image source, OcrLine line, int padX, int padY) {
    return _cropRect(
      source,
      line.x - padX,
      line.y - padY,
      line.x + _maxInt(line.width, 1) + padX,
      line.y + _maxInt(line.height, 1) + padY,
    );
  }

  img.Image? _cropRect(
      img.Image source, int left, int top, int right, int bottom) {
    final clampedLeft = _clampInt(left, 0, source.width);
    final clampedTop = _clampInt(top, 0, source.height);
    final clampedRight = _clampInt(right, 0, source.width);
    final clampedBottom = _clampInt(bottom, 0, source.height);
    if (clampedRight - clampedLeft < 8 || clampedBottom - clampedTop < 8) {
      return null;
    }
    return img.copyCrop(
      source,
      x: clampedLeft,
      y: clampedTop,
      width: clampedRight - clampedLeft,
      height: clampedBottom - clampedTop,
    );
  }

  int _clampInt(int value, int minValue, int maxValue) {
    if (value < minValue) {
      return minValue;
    }
    if (value > maxValue) {
      return maxValue;
    }
    return value;
  }

  int _maxInt(int left, int right) {
    return left > right ? left : right;
  }
}

class _ParsedNumber {
  const _ParsedNumber({
    required this.value,
    required this.normalized,
  });

  final double value;
  final String normalized;
}

class _ExtractionCandidate {
  const _ExtractionCandidate({
    required this.value,
    required this.line,
    required this.normalized,
    required this.order,
  });

  final double value;
  final OcrLine line;
  final String normalized;
  final int order;
}

class _CandidatePair {
  const _CandidatePair({
    required this.left,
    required this.right,
    required this.score,
  });

  final _ExtractionCandidate left;
  final _ExtractionCandidate right;
  final double score;
}

class _RescuedWeight {
  const _RescuedWeight({
    required this.value,
    required this.score,
  });

  final double value;
  final double score;
}
