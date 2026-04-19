import 'dart:io';

import 'package:egg_ocr/egg_ocr.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../shared/benchmark_runner.dart';
import '../../../shared/desktop_page_header.dart';
import '../../../shared/portrait_image.dart';

class BenchmarkPage extends ConsumerStatefulWidget {
  const BenchmarkPage({super.key});

  @override
  ConsumerState<BenchmarkPage> createState() => _BenchmarkPageState();
}

class _BenchmarkPageState extends ConsumerState<BenchmarkPage> {
  OcrBenchmarkReport? _report;
  Object? _error;
  bool _running = false;

  Future<void> _run() async {
    setState(() {
      _running = true;
      _error = null;
    });
    try {
      final report = await ref.read(benchmarkRunnerProvider).run();
      setState(() => _report = report);
    } catch (error) {
      setState(() => _error = error);
    } finally {
      if (mounted) {
        setState(() => _running = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final useDesktopFrame = Platform.isWindows;
    final showAppBar =
        !useDesktopFrame && MediaQuery.sizeOf(context).width >= 320;
    final body = ListView(
      padding: const EdgeInsets.all(20),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('基准说明',
                    style: TextStyle(fontWeight: FontWeight.w700)),
                const SizedBox(height: 8),
                const Text('会用内置样本图真实调用当前平台 OCR，再跑数值提取和候选匹配，统计命中情况。'),
                const SizedBox(height: 8),
                Text('当前引擎：${currentOcrEngineLabel()}'),
                const SizedBox(height: 12),
                FilledButton.icon(
                  onPressed: _running ? null : _run,
                  icon: const Icon(Icons.science_outlined),
                  label: Text(_running ? '运行中...' : '开始基准测试'),
                ),
              ],
            ),
          ),
        ),
        if (_error != null) ...[
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Text('运行失败：$_error'),
            ),
          ),
        ],
        if (_report != null) ...[
          const SizedBox(height: 16),
          Card(
            child: ListTile(
              title: const Text('总览'),
              subtitle: Text(
                '通过 ${_report!.passedCount} / ${_report!.totalCount}\n'
                '时间：${_report!.generatedAt.toLocal()}',
              ),
            ),
          ),
          const SizedBox(height: 16),
          for (final item in _report!.cases) ...[
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            item.config.caseName,
                            style: const TextStyle(fontWeight: FontWeight.w700),
                          ),
                        ),
                        Chip(
                          label: Text(item.passed ? 'PASS' : 'FAIL'),
                          backgroundColor: item.passed
                              ? Colors.green.shade100
                              : Colors.red.shade100,
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '预期：${item.config.expectedMeasurements.map((e) => '(${e.size.toStringAsFixed(3)}, ${e.weight.toStringAsFixed(3)})').join(', ')}',
                    ),
                    Text(
                      '实际：${item.measurements.map((e) => '(${e.heightInMeters.toStringAsFixed(3)}, ${e.weightInKg.toStringAsFixed(3)})').join(', ')}',
                    ),
                    const SizedBox(height: 12),
                    Text('OCR 行数：${item.ocrDocument.lines.length}'),
                    for (final line in item.ocrDocument.lines.take(8))
                      Text(
                        '• ${line.text} @ (${line.bounds.left.toStringAsFixed(0)}, ${line.bounds.top.toStringAsFixed(0)})',
                      ),
                    const SizedBox(height: 12),
                    for (final entry in item.entries)
                      if (entry.candidates.isNotEmpty)
                        ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading: PortraitImage(
                            portraitKey: entry.candidates.first.portraitKey,
                            label: entry.candidates.first.petName,
                            size: 56,
                          ),
                          title: Text(entry.candidates.first.petName),
                          subtitle: Text(
                            'Top1 概率 ${entry.candidates.first.probability.toStringAsFixed(2)}%\n'
                            '${entry.candidates.first.heightRangeLabel} | ${entry.candidates.first.weightRangeLabel}',
                          ),
                        ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
          ],
        ],
      ],
    );
    return Scaffold(
      appBar: showAppBar ? AppBar(title: const Text('OCR 基准测试')) : null,
      body: useDesktopFrame
          ? DesktopPageFrame(
              title: 'OCR 基准测试',
              currentRoute: '/benchmark',
              leading: IconButton(
                tooltip: '返回',
                onPressed: () {
                  if (context.canPop()) {
                    context.pop();
                  } else {
                    context.go('/settings');
                  }
                },
                icon: const Icon(Icons.arrow_back_rounded),
              ),
              child: body,
            )
          : body,
    );
  }
}
