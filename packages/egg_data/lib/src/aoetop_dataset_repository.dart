import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:egg_core/egg_core.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

class AoetopDatasetRepository implements DatasetRepository {
  AoetopDatasetRepository({
    Dio? dio,
    Duration staleAfter = const Duration(hours: 24),
  })  : _dio = dio ?? Dio(),
        _staleAfter = staleAfter;

  static const datasetUrl = 'https://rocom.aoe.top/data/Pets.json';
  static const _cacheFileName = 'aoe-pets.json';

  final Dio _dio;
  final Duration _staleAfter;

  @override
  Future<DatasetSnapshot> load() async {
    final file = await _cacheFile();
    if (await file.exists()) {
      final content = await file.readAsString();
      final dataset = PetsDataset.fromJson(
        jsonDecode(content) as List<dynamic>,
      );
      final updatedAt = await file.lastModified();
      return DatasetSnapshot(
        dataset: dataset,
        updatedAt: updatedAt,
        fromCache: true,
      );
    }
    return refresh();
  }

  @override
  Future<DatasetSnapshot> refresh() async {
    final response = await _dio.get<String>(datasetUrl);
    final body = response.data ?? '[]';
    final file = await _cacheFile();
    await file.parent.create(recursive: true);
    await file.writeAsString(body, flush: true);
    final dataset = PetsDataset.fromJson(jsonDecode(body) as List<dynamic>);
    final updatedAt = await file.lastModified();
    return DatasetSnapshot(
      dataset: dataset,
      updatedAt: updatedAt,
      fromCache: false,
    );
  }

  Future<bool> isStale() async {
    final file = await _cacheFile();
    if (!await file.exists()) {
      return true;
    }
    final age = DateTime.now().difference(await file.lastModified());
    return age > _staleAfter;
  }

  Future<File> _cacheFile() async {
    final root = await getApplicationSupportDirectory();
    return File(p.join(root.path, 'dataset', _cacheFileName));
  }
}
