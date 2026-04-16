import 'dart:math' as math;
import 'dart:ui';

import 'models.dart';

class MeasurementExtractor {
  const MeasurementExtractor();

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
        ),
      );
    }

    results.sort((left, right) {
      if ((left.anchor.dy - right.anchor.dy).abs() <= 12) {
        return left.anchor.dx.compareTo(right.anchor.dx);
      }
      return left.anchor.dy.compareTo(right.anchor.dy);
    });

    return _dedupe(results);
  }

  _ParsedNumber? _parseNumber(String input) {
    final normalized = _normalizeNumericText(input);
    final match = RegExp(r'(?:\d+\.\d+|\.\d+|\d+)').firstMatch(normalized);
    if (match == null) {
      return null;
    }
    var valueText = match.group(0)!;
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
        return '5'.codeUnitAt(0);
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
