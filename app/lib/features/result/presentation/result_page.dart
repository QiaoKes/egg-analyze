import 'package:egg_core/egg_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../shared/portrait_image.dart';
import '../../../shared/providers.dart';

class ResultPage extends ConsumerWidget {
  const ResultPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final result = ref.watch(currentAnalysisResultProvider);
    if (result == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('分析结果')),
        body: Center(
          child: FilledButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('先返回首页导入图片'),
          ),
        ),
      );
    }

    return Scaffold(
      appBar: AppBar(title: const Text('分析结果')),
      body: LayoutBuilder(
        builder: (context, constraints) {
          final wide = constraints.maxWidth >= 720;
          final preview = _SourcePreviewCard(result: result);
          final results = _ResultPanel(result: result);

          if (!wide) {
            return ListView(
              padding: const EdgeInsets.all(20),
              children: [
                preview,
                const SizedBox(height: 20),
                SizedBox(
                  height: 560,
                  child: results,
                ),
              ],
            );
          }

          final contentHeight = constraints.maxHeight - 40;
          return Padding(
            padding: const EdgeInsets.all(20),
            child: SizedBox(
              height: contentHeight,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    flex: 5,
                    child: SizedBox(
                      height: contentHeight,
                      child: preview,
                    ),
                  ),
                  const SizedBox(width: 20),
                  Expanded(
                    flex: 6,
                    child: SizedBox(
                      height: contentHeight,
                      child: results,
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}

class _SourcePreviewCard extends StatelessWidget {
  const _SourcePreviewCard({required this.result});

  final AnalysisResult result;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('图片预览', style: TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(height: 12),
            Text('来源：${result.sourceLabel}'),
            const SizedBox(height: 8),
            Text('分析时间：${result.analyzedAt.toLocal()}'),
            const SizedBox(height: 16),
            Expanded(
              child: ClipRRect(
                borderRadius: BorderRadius.circular(16),
                child: Container(
                  width: double.infinity,
                  color: const Color(0xFFF4F6FA),
                  alignment: Alignment.center,
                  child: Image.memory(
                    result.sourceBytes,
                    fit: BoxFit.contain,
                    errorBuilder: (_, __, ___) => const SizedBox.shrink(),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ResultPanel extends StatefulWidget {
  const _ResultPanel({required this.result});

  final AnalysisResult result;

  @override
  State<_ResultPanel> createState() => _ResultPanelState();
}

class _ResultPanelState extends State<_ResultPanel> {
  late final ScrollController _scrollController;

  @override
  void initState() {
    super.initState();
    _scrollController = ScrollController();
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Scrollbar(
          controller: _scrollController,
          thumbVisibility: true,
          child: ListView(
            controller: _scrollController,
            children: [
              const Text('识别结果', style: TextStyle(fontWeight: FontWeight.w700)),
              const SizedBox(height: 12),
              if (widget.result.entries.isEmpty)
                const Padding(
                  padding: EdgeInsets.only(bottom: 8),
                  child: Text('未提取到有效的身高/体重。'),
                ),
              for (var i = 0; i < widget.result.entries.length; i++) ...[
                _MeasurementCard(index: i + 1, entry: widget.result.entries[i]),
                const SizedBox(height: 16),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _MeasurementCard extends StatelessWidget {
  const _MeasurementCard({
    required this.index,
    required this.entry,
  });

  final int index;
  final AnalysisEntry entry;

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 0,
      color: const Color(0xFFF7F9FD),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('蛋 $index',
                style: const TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(height: 12),
            Wrap(
              spacing: 16,
              runSpacing: 8,
              children: [
                _MetricChip(
                  label: '身高',
                  value:
                      '${entry.measurement.heightInMeters.toStringAsFixed(3)} m',
                ),
                _MetricChip(
                  label: '体重',
                  value:
                      '${entry.measurement.weightInKg.toStringAsFixed(3)} kg',
                ),
              ],
            ),
            const SizedBox(height: 16),
            for (final candidate in entry.candidates) ...[
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: PortraitImage(
                  portraitKey: candidate.portraitKey,
                  label: candidate.petName,
                ),
                title: Text('${candidate.petName} (#${candidate.petId})'),
                subtitle: Text(
                  '概率 ${candidate.probability.toStringAsFixed(2)}% | '
                  '身高 ${candidate.heightRangeLabel} | '
                  '体重 ${candidate.weightRangeLabel}\n'
                  '${candidate.matchLabel}'
                  '${candidate.hatchLabel == null ? '' : ' | 孵化 ${candidate.hatchLabel}'}',
                ),
              ),
              const Divider(height: 1),
            ],
            if (entry.candidates.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 8),
                child: Text('没有命中候选，通常说明 OCR 提取到了数值，但未落入公开区间。'),
              ),
          ],
        ),
      ),
    );
  }
}

class _MetricChip extends StatelessWidget {
  const _MetricChip({
    required this.label,
    required this.value,
  });

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xFFE8EEF9),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label, style: Theme.of(context).textTheme.labelMedium),
            Text(value, style: Theme.of(context).textTheme.titleMedium),
          ],
        ),
      ),
    );
  }
}
