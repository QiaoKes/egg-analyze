import 'dart:math' as math;

import 'models.dart';

class MatchingEngine {
  MatchingEngine(PetsDataset dataset)
      : _chains = _buildChains(dataset),
        measurementPriors = _buildPriors(dataset);

  final List<_ChainEntry> _chains;
  final MeasurementPriors measurementPriors;

  List<Candidate> search({
    required double heightInCentimeters,
    required double weightInKg,
    int limit = 5,
  }) {
    final matches = <Candidate>[];
    for (final chain in _chains) {
      Candidate? best;
      for (final variant in chain.variants) {
        final evaluation = _evaluateVariantMatch(
          heightInCentimeters: heightInCentimeters,
          weightInKg: weightInKg,
          variant: variant,
        );
        if (evaluation == null) {
          continue;
        }

        final candidate = Candidate(
          petId: chain.rootPet.id,
          petName: _petDisplayName(chain.rootPet),
          portraitKey: chain.rootPet.name.trim(),
          probability: 0,
          heightRangeLabel: _formatRange(
            variant.heightRange,
            variant.hasHeight,
            'cm',
          ),
          weightRangeLabel: _formatRange(
            variant.weightRange,
            variant.hasWeight,
            'kg',
          ),
          matchLabel: _buildMatchLabel(
            root: chain.rootPet,
            sourcePetName: variant.sourcePetName,
            matchedMetricCount: evaluation.matchedMetricCount,
          ),
          matchScore: evaluation.score,
          implemented: chain.rootPet.implemented,
          hatchLabel: _formatHatchLabel(variant.hatchSeconds),
        );

        if (best == null ||
            candidate.matchScore < best.matchScore ||
            (candidate.matchScore == best.matchScore &&
                candidate.petName.compareTo(best.petName) < 0)) {
          best = candidate;
        }
      }

      if (best != null) {
        matches.add(best);
      }
    }

    final visibleMatches = matches.where((item) => item.implemented).toList();
    final ranked = visibleMatches.isNotEmpty ? visibleMatches : matches;
    _assignProbabilities(ranked);
    ranked.sort((left, right) {
      if (left.probability != right.probability) {
        return right.probability.compareTo(left.probability);
      }
      if (left.matchScore != right.matchScore) {
        return left.matchScore.compareTo(right.matchScore);
      }
      return left.petName.compareTo(right.petName);
    });

    final uniqueByName = <Candidate>[];
    final seenNames = <String>{};
    for (final item in ranked) {
      final normalizedName = item.petName.trim();
      if (!seenNames.add(normalizedName)) {
        continue;
      }
      uniqueByName.add(item);
    }

    if (uniqueByName.length <= limit) {
      return uniqueByName;
    }
    return uniqueByName.take(limit).toList(growable: false);
  }

  static MeasurementPriors _buildPriors(PetsDataset dataset) {
    Range? height;
    Range? weight;
    for (final pet in dataset.pets) {
      for (final variant in _extractBreedingVariants(pet)) {
        final heightRange = _buildRange(
          variant.heightLow,
          variant.heightHigh,
          transform: (value) => value / 100,
        );
        final weightRange = _buildRange(
          variant.weightLow,
          variant.weightHigh,
          transform: (value) => value / 1000,
        );
        if (heightRange != null) {
          height = height == null
              ? heightRange
              : Range(
                  min: math.min(height.min, heightRange.min),
                  max: math.max(height.max, heightRange.max),
                );
        }
        if (weightRange != null) {
          weight = weight == null
              ? weightRange
              : Range(
                  min: math.min(weight.min, weightRange.min),
                  max: math.max(weight.max, weightRange.max),
                );
        }
      }
    }

    return MeasurementPriors(
      diameter: height ?? const Range(min: 0.03, max: 1.2),
      weight: weight ?? const Range(min: 0.03, max: 300),
    );
  }

  static List<_ChainEntry> _buildChains(PetsDataset dataset) {
    final petsById = {for (final pet in dataset.pets) pet.id: pet};
    final rootCache = <int, int>{};
    final grouped = <int, _ChainEntry>{};

    for (final pet in dataset.pets) {
      final rootId = _resolveRootId(pet, petsById, rootCache);
      final rootPet = petsById[rootId] ?? pet;
      final entry = grouped.putIfAbsent(
        rootId,
        () => _ChainEntry(rootPet: rootPet),
      );
      entry.memberCount++;

      for (final variant in _extractBreedingVariants(pet)) {
        final variantEntry = _VariantEntry.fromPetVariant(pet, variant);
        if (variantEntry == null) {
          continue;
        }
        if (entry.variantKeys.add(variantEntry.key)) {
          entry.variants.add(variantEntry);
        }
      }
    }

    final chains = grouped.values
        .where((entry) => entry.variants.isNotEmpty)
        .toList(growable: false);
    chains.sort(
      (left, right) => _petDisplayName(left.rootPet)
          .compareTo(_petDisplayName(right.rootPet)),
    );
    return chains;
  }

  static int _resolveRootId(
    Pet pet,
    Map<int, Pet> petsById,
    Map<int, int> cache,
  ) {
    final cached = cache[pet.id];
    if (cached != null) {
      return cached;
    }

    final visited = <int>[];
    final seen = <int>{};
    var current = pet;
    while (current.evolvesFromId != null) {
      if (!seen.add(current.id)) {
        break;
      }
      visited.add(current.id);
      final parent = petsById[current.evolvesFromId];
      if (parent == null) {
        break;
      }
      current = parent;
    }

    cache[current.id] = current.id;
    for (final id in visited) {
      cache[id] = current.id;
    }
    return current.id;
  }

  static List<BreedingVariant> _extractBreedingVariants(Pet pet) {
    final breeding = pet.breeding;
    if (breeding == null) {
      return const [];
    }
    final variants = breeding.variants.isNotEmpty
        ? breeding.variants
        : <BreedingVariant>[breeding];
    return variants
        .where(
          (variant) =>
              variant.heightLow != null ||
              variant.heightHigh != null ||
              variant.weightLow != null ||
              variant.weightHigh != null,
        )
        .toList(growable: false);
  }

  static Range? _buildRange(
    double? low,
    double? high, {
    required double Function(double value) transform,
  }) {
    if (low == null && high == null) {
      return null;
    }
    final left = transform(low ?? high!);
    final right = transform(high ?? low!);
    return Range(
      min: math.min(left, right),
      max: math.max(left, right),
    );
  }

  _MatchEvaluation? _evaluateVariantMatch({
    required double heightInCentimeters,
    required double weightInKg,
    required _VariantEntry variant,
  }) {
    var score = 0.0;
    var matchedMetricCount = 0;

    if (variant.hasHeight) {
      final heightScore =
          _evaluateMetric(heightInCentimeters, variant.heightRange);
      if (heightScore == null) {
        return null;
      }
      score += heightScore;
      matchedMetricCount++;
    }

    if (variant.hasWeight) {
      final weightScore = _evaluateMetric(weightInKg, variant.weightRange);
      if (weightScore == null) {
        return null;
      }
      score += weightScore;
      matchedMetricCount++;
    }

    if (matchedMetricCount == 0) {
      return null;
    }

    return _MatchEvaluation(
      score: score,
      matchedMetricCount: matchedMetricCount,
    );
  }

  double? _evaluateMetric(double value, Range range) {
    if (value < range.min || value > range.max) {
      return null;
    }
    final center = (range.min + range.max) / 2;
    final span = math.max(range.max - range.min, 1);
    final centerPenalty = (value - center).abs() / math.max(span / 2, 1);
    final spanPenalty = (span / math.max(center, 1)) * 0.35;
    return centerPenalty + spanPenalty;
  }

  static void _assignProbabilities(List<Candidate> items) {
    if (items.isEmpty) {
      return;
    }

    final weights = <double>[];
    var sum = 0.0;
    for (final item in items) {
      final weight = math.exp(-item.matchScore);
      weights.add(weight);
      sum += weight;
    }
    if (sum <= 0) {
      return;
    }

    for (var i = 0; i < items.length; i++) {
      items[i] = Candidate(
        petId: items[i].petId,
        petName: items[i].petName,
        portraitKey: items[i].portraitKey,
        probability: double.parse(
          ((weights[i] / sum) * 100).toStringAsFixed(2),
        ),
        heightRangeLabel: items[i].heightRangeLabel,
        weightRangeLabel: items[i].weightRangeLabel,
        matchLabel: items[i].matchLabel,
        matchScore: items[i].matchScore,
        implemented: items[i].implemented,
        hatchLabel: items[i].hatchLabel,
      );
    }
  }

  static String _petDisplayName(Pet pet) {
    return pet.localizedName.isNotEmpty ? pet.localizedName : pet.name;
  }

  static String _buildMatchLabel({
    required Pet root,
    required String sourcePetName,
    required int matchedMetricCount,
  }) {
    final prefix = matchedMetricCount == 1 ? '单维命中' : '双维命中';
    final rootName = _petDisplayName(root);
    if (sourcePetName.isEmpty || sourcePetName == rootName) {
      return '$prefix 最低阶段';
    }
    return '$prefix $sourcePetName，结果回落到最低阶段';
  }

  static String _formatRange(Range range, bool hasValue, String unit) {
    if (!hasValue) {
      return '暂无数据';
    }
    if ((range.min - range.max).abs() <= 1e-9) {
      return '${_formatNumber(range.min)}$unit';
    }
    return '${_formatNumber(range.min)}-${_formatNumber(range.max)}$unit';
  }

  static String _formatNumber(double value) {
    if ((value - value.roundToDouble()).abs() <= 1e-9) {
      return value.round().toString();
    }
    return value.toStringAsFixed(2);
  }

  static String? _formatHatchLabel(int? seconds) {
    if (seconds == null || seconds <= 0) {
      return null;
    }
    if (seconds % 86400 == 0) {
      return '${seconds ~/ 86400} 天';
    }
    final hours = seconds / 3600;
    if ((hours - hours.roundToDouble()).abs() <= 1e-9) {
      return '${hours.round()} 小时';
    }
    return '${hours.toStringAsFixed(1)} 小时';
  }
}

class _ChainEntry {
  _ChainEntry({required this.rootPet});

  final Pet rootPet;
  final Set<String> variantKeys = <String>{};
  final List<_VariantEntry> variants = <_VariantEntry>[];
  int memberCount = 0;
}

class _VariantEntry {
  const _VariantEntry({
    required this.key,
    required this.sourcePetName,
    required this.heightRange,
    required this.weightRange,
    required this.hasHeight,
    required this.hasWeight,
    required this.hatchSeconds,
  });

  final String key;
  final String sourcePetName;
  final Range heightRange;
  final Range weightRange;
  final bool hasHeight;
  final bool hasWeight;
  final int? hatchSeconds;

  static _VariantEntry? fromPetVariant(Pet pet, BreedingVariant variant) {
    final heightRange = MatchingEngine._buildRange(
      variant.heightLow,
      variant.heightHigh,
      transform: (value) => value,
    );
    final weightRange = MatchingEngine._buildRange(
      variant.weightLow,
      variant.weightHigh,
      transform: (value) => value / 1000,
    );
    if (heightRange == null && weightRange == null) {
      return null;
    }

    return _VariantEntry(
      key: [
        pet.id,
        variant.id ?? 'na',
        variant.heightLow ?? 'na',
        variant.heightHigh ?? 'na',
        variant.weightLow ?? 'na',
        variant.weightHigh ?? 'na',
        variant.hatchData ?? 'na',
      ].join(':'),
      sourcePetName: pet.localizedName,
      heightRange: heightRange ?? const Range(min: 0, max: 0),
      weightRange: weightRange ?? const Range(min: 0, max: 0),
      hasHeight: heightRange != null,
      hasWeight: weightRange != null,
      hatchSeconds: variant.hatchData,
    );
  }
}

class _MatchEvaluation {
  const _MatchEvaluation({
    required this.score,
    required this.matchedMetricCount,
  });

  final double score;
  final int matchedMetricCount;
}
