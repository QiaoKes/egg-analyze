import 'dart:typed_data';
import 'dart:ui';

class OcrDocument {
  const OcrDocument({required this.lines});

  final List<OcrLine> lines;
}

class OcrLine {
  const OcrLine({
    required this.text,
    required this.bounds,
  });

  final String text;
  final Rect bounds;

  int get x => bounds.left.round();
  int get y => bounds.top.round();
  int get width => bounds.width.round();
  int get height => bounds.height.round();
}

class Range {
  const Range({
    required this.min,
    required this.max,
  });

  final double min;
  final double max;
}

class MeasurementPriors {
  const MeasurementPriors({
    required this.diameter,
    required this.weight,
  });

  final Range diameter;
  final Range weight;
}

class Measurement {
  const Measurement({
    required this.heightInMeters,
    required this.weightInKg,
    required this.anchorText,
    required this.anchor,
    required this.rawLines,
    this.eggName,
    this.weightText,
    this.weightBounds,
  });

  final double heightInMeters;
  final double weightInKg;
  final String anchorText;
  final Offset anchor;
  final List<OcrLine> rawLines;
  final String? eggName;
  final String? weightText;
  final Rect? weightBounds;
}

class AnalysisEntry {
  const AnalysisEntry({
    required this.measurement,
    required this.candidates,
  });

  final Measurement measurement;
  final List<Candidate> candidates;
}

class AnalysisResult {
  const AnalysisResult({
    required this.entries,
    required this.ocrDocument,
    required this.sourceBytes,
    required this.sourceLabel,
    required this.analyzedAt,
  });

  final List<AnalysisEntry> entries;
  final OcrDocument ocrDocument;
  final Uint8List sourceBytes;
  final String sourceLabel;
  final DateTime analyzedAt;
}

class Candidate {
  const Candidate({
    required this.petId,
    required this.petName,
    required this.portraitKey,
    required this.probability,
    required this.heightRangeLabel,
    required this.weightRangeLabel,
    required this.matchLabel,
    required this.matchScore,
    required this.implemented,
    required this.hatchLabel,
  });

  final int petId;
  final String petName;
  final String portraitKey;
  final double probability;
  final String heightRangeLabel;
  final String weightRangeLabel;
  final String matchLabel;
  final double matchScore;
  final bool implemented;
  final String? hatchLabel;
}

class PetsDataset {
  const PetsDataset({
    required this.pets,
  });

  final List<Pet> pets;

  factory PetsDataset.fromJson(List<dynamic> json) {
    return PetsDataset(
      pets: json
          .whereType<Map<String, dynamic>>()
          .map(Pet.fromJson)
          .toList(growable: false),
    );
  }
}

class Pet {
  const Pet({
    required this.id,
    required this.name,
    required this.localizedName,
    required this.implemented,
    required this.evolvesFromId,
    required this.breeding,
  });

  final int id;
  final String name;
  final String localizedName;
  final bool implemented;
  final int? evolvesFromId;
  final BreedingInfo? breeding;

  factory Pet.fromJson(Map<String, dynamic> json) {
    final localized = (json['localized'] as Map<String, dynamic>?)?['zh']
        as Map<String, dynamic>?;
    return Pet(
      id: json['id'] as int,
      name: (json['name'] as String?)?.trim() ?? '',
      localizedName: (localized?['name'] as String?)?.trim() ?? '',
      implemented: json['implemented'] as bool? ?? false,
      evolvesFromId: json['evolves_from_id'] as int?,
      breeding: json['breeding'] is Map<String, dynamic>
          ? BreedingInfo.fromJson(json['breeding'] as Map<String, dynamic>)
          : null,
    );
  }
}

class BreedingInfo extends BreedingVariant {
  const BreedingInfo({
    required super.id,
    required super.petId,
    required super.name,
    required super.modelId,
    required super.hatchData,
    required super.weightLow,
    required super.weightHigh,
    required super.heightLow,
    required super.heightHigh,
    required this.variants,
  });

  final List<BreedingVariant> variants;

  factory BreedingInfo.fromJson(Map<String, dynamic> json) {
    final base = BreedingVariant.fromJson(json);
    return BreedingInfo(
      id: base.id,
      petId: base.petId,
      name: base.name,
      modelId: base.modelId,
      hatchData: base.hatchData,
      weightLow: base.weightLow,
      weightHigh: base.weightHigh,
      heightLow: base.heightLow,
      heightHigh: base.heightHigh,
      variants: ((json['variants'] as List<dynamic>?) ?? const [])
          .whereType<Map<String, dynamic>>()
          .map(BreedingVariant.fromJson)
          .toList(growable: false),
    );
  }
}

class BreedingVariant {
  const BreedingVariant({
    required this.id,
    required this.petId,
    required this.name,
    required this.modelId,
    required this.hatchData,
    required this.weightLow,
    required this.weightHigh,
    required this.heightLow,
    required this.heightHigh,
  });

  final int? id;
  final int? petId;
  final String? name;
  final int? modelId;
  final int? hatchData;
  final double? weightLow;
  final double? weightHigh;
  final double? heightLow;
  final double? heightHigh;

  factory BreedingVariant.fromJson(Map<String, dynamic> json) {
    double? asDouble(Object? value) => value is num ? value.toDouble() : null;

    return BreedingVariant(
      id: json['id'] as int?,
      petId: json['pet_id'] as int?,
      name: json['name'] as String?,
      modelId: json['model_id'] as int?,
      hatchData: json['hatch_data'] as int?,
      weightLow: asDouble(json['weight_low']),
      weightHigh: asDouble(json['weight_high']),
      heightLow: asDouble(json['height_low']),
      heightHigh: asDouble(json['height_high']),
    );
  }
}

class DatasetSnapshot {
  const DatasetSnapshot({
    required this.dataset,
    required this.updatedAt,
    required this.fromCache,
  });

  final PetsDataset dataset;
  final DateTime updatedAt;
  final bool fromCache;
}

class RecentAnalysisRecord {
  const RecentAnalysisRecord({
    required this.id,
    required this.label,
    required this.createdAt,
  });

  final String id;
  final String label;
  final DateTime createdAt;

  Map<String, dynamic> toJson() => {
        'id': id,
        'label': label,
        'createdAt': createdAt.toIso8601String(),
      };

  factory RecentAnalysisRecord.fromJson(Map<String, dynamic> json) {
    return RecentAnalysisRecord(
      id: json['id'] as String,
      label: json['label'] as String,
      createdAt: DateTime.parse(json['createdAt'] as String),
    );
  }
}
