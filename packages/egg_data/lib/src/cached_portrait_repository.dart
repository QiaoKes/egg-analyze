import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:image/image.dart' as img;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

class CachedPortraitRepository {
  CachedPortraitRepository({Dio? dio}) : _dio = dio ?? Dio();

  static const atlasBaseUrl = 'https://rocom.aoe.top/assets/webp/friends';

  final Dio _dio;
  final Map<String, Future<Uint8List?>> _pendingLoads = {};

  Future<Uint8List?> loadPortraitBytes(String portraitKey) async {
    final normalized = _normalizeKey(portraitKey);
    if (normalized.isEmpty) {
      return null;
    }

    final file = await _cacheFile(normalized);
    if (await file.exists()) {
      return file.readAsBytes();
    }

    final pending = _pendingLoads[normalized];
    if (pending != null) {
      return pending;
    }

    final future = _loadAndCachePortraitBytes(normalized);
    _pendingLoads[normalized] = future;
    try {
      return await future;
    } finally {
      if (identical(_pendingLoads[normalized], future)) {
        _pendingLoads.remove(normalized);
      }
    }
  }

  Future<void> preloadPortraits(
    Iterable<String> portraitKeys, {
    int batchSize = 2,
  }) async {
    final uniqueKeys = portraitKeys
        .map(_normalizeKey)
        .where((value) => value.isNotEmpty)
        .toSet()
        .toList(growable: false);
    if (uniqueKeys.isEmpty) {
      return;
    }

    for (var index = 0; index < uniqueKeys.length; index += batchSize) {
      final batch = uniqueKeys.skip(index).take(batchSize);
      await Future.wait(batch.map(loadPortraitBytes));
    }
  }

  Future<void> clearCache() async {
    final root = await getApplicationSupportDirectory();
    final dir = Directory(p.join(root.path, 'friend-atlas'));
    if (await dir.exists()) {
      await dir.delete(recursive: true);
    }
  }

  Future<File> _cacheFile(String normalized) async {
    final root = await getApplicationSupportDirectory();
    return File(p.join(root.path, 'friend-atlas', 'JL_$normalized.png'));
  }

  Future<File> _legacyCacheFile(String normalized) async {
    final root = await getApplicationSupportDirectory();
    return File(p.join(root.path, 'friend-atlas', 'JL_$normalized.webp'));
  }

  String _normalizeKey(String value) {
    return value
        .trim()
        .replaceFirst(RegExp(r'^JL_'), '')
        .replaceFirst('.webp', '');
  }

  Future<Uint8List?> _loadAndCachePortraitBytes(String normalized) async {
    final file = await _cacheFile(normalized);
    if (await file.exists()) {
      return file.readAsBytes();
    }

    final legacyFile = await _legacyCacheFile(normalized);
    if (await legacyFile.exists()) {
      final bytes = await legacyFile.readAsBytes();
      final normalizedBytes = await _normalizePortraitBytes(bytes);
      if (normalizedBytes == null) {
        return null;
      }

      await file.parent.create(recursive: true);
      await file.writeAsBytes(normalizedBytes, flush: true);
      return normalizedBytes;
    }

    final response = await _dio.get<List<int>>(
      '$atlasBaseUrl/JL_$normalized.webp',
      options: Options(responseType: ResponseType.bytes),
    );
    final contentType =
        (response.headers.value(Headers.contentTypeHeader) ?? '').toLowerCase();
    if (!contentType.startsWith('image/')) {
      return null;
    }

    final normalizedBytes = await _normalizePortraitBytes(
      Uint8List.fromList(response.data ?? const <int>[]),
    );
    if (normalizedBytes == null) {
      return null;
    }

    await file.parent.create(recursive: true);
    await file.writeAsBytes(normalizedBytes, flush: true);
    return normalizedBytes;
  }
}

Future<Uint8List?> _normalizePortraitBytes(Uint8List bytes) {
  return compute(_normalizePortraitBytesOnWorker, bytes);
}

Uint8List? _normalizePortraitBytesOnWorker(Uint8List bytes) {
  final decoded = img.decodeImage(bytes);
  if (decoded == null) {
    return null;
  }

  final trimmed = _trimTransparentBounds(decoded);
  return Uint8List.fromList(img.encodePng(trimmed));
}

img.Image _trimTransparentBounds(img.Image source) {
  if (!source.hasAlpha) {
    return source;
  }

  var left = source.width;
  var top = source.height;
  var right = -1;
  var bottom = -1;

  for (final pixel in source) {
    if (pixel.a <= 0) {
      continue;
    }
    if (pixel.x < left) {
      left = pixel.x;
    }
    if (pixel.y < top) {
      top = pixel.y;
    }
    if (pixel.x > right) {
      right = pixel.x;
    }
    if (pixel.y > bottom) {
      bottom = pixel.y;
    }
  }

  if (right < left || bottom < top) {
    return source;
  }

  final width = right - left + 1;
  final height = bottom - top + 1;
  if (width == source.width && height == source.height) {
    return source;
  }

  return img.copyCrop(
    source,
    x: left,
    y: top,
    width: width,
    height: height,
  );
}
