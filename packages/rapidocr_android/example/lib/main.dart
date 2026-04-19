import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:rapidocr_android/rapidocr_android.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatefulWidget {
  const MyApp({super.key});

  @override
  State<MyApp> createState() => _MyAppState();
}

class _MyAppState extends State<MyApp> {
  String _status = 'Idle';
  final _rapidocrAndroidPlugin = RapidOcrAndroid();

  @override
  void initState() {
    super.initState();
    initPlatformState();
  }

  Future<void> initPlatformState() async {
    try {
      await _rapidocrAndroidPlugin.initialize();
      if (!mounted) {
        return;
      }
      setState(() {
        _status = 'RapidOCR initialized';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _status = 'Initialization failed: $error';
      });
    }
  }

  Future<void> runSampleRecognition() async {
    try {
      final document = await _rapidocrAndroidPlugin.recognize(Uint8List(0));
      if (!mounted) {
        return;
      }
      setState(() {
        _status = 'Recognized ${document.lines.length} lines';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _status = 'Recognition failed: $error';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: const Text('Plugin example app')),
        body: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(_status),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: runSampleRecognition,
                child: const Text('Run sample'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
