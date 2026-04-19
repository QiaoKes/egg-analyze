import 'dart:typed_data';

import 'package:egg_core/egg_core.dart';
import 'package:plugin_platform_interface/plugin_platform_interface.dart';

import 'rapidocr_android.dart';
import 'rapidocr_android_method_channel.dart';

abstract class RapidocrAndroidPlatform extends PlatformInterface {
  RapidocrAndroidPlatform() : super(token: _token);

  static final Object _token = Object();

  static RapidocrAndroidPlatform _instance = MethodChannelRapidocrAndroid();

  static RapidocrAndroidPlatform get instance => _instance;

  static set instance(RapidocrAndroidPlatform instance) {
    PlatformInterface.verifyToken(instance, _token);
    _instance = instance;
  }

  Future<void> initialize();

  Future<OcrDocument> recognize(
    Uint8List imageBytes, {
    required RapidOcrMode mode,
  });

  Future<List<OcrDocument>> recognizeBatch(
    List<Uint8List> imageBytesList, {
    required RapidOcrMode mode,
  });

  Future<void> dispose();
}
