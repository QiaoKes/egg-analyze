import 'dart:io';

import 'package:egg_core/egg_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../shared/desktop_page_header.dart';
import '../../../shared/desktop_window_actions.dart';
import '../../../shared/portrait_image.dart';
import '../../../shared/providers.dart';
import '../../../shared/screenshot_analysis_flow.dart';

class ResultPage extends ConsumerStatefulWidget {
  const ResultPage({super.key});

  @override
  ConsumerState<ResultPage> createState() => _ResultPageState();
}

class _ResultPageState extends ConsumerState<ResultPage> {
  bool _captureBusy = false;

  Future<void> _captureAgain() async {
    if (_captureBusy || !supportsRegionCapture) {
      return;
    }
    setState(() => _captureBusy = true);
    try {
      await captureRegionAndAnalyze(context, ref);
    } finally {
      if (mounted) {
        setState(() => _captureBusy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final result = ref.watch(currentAnalysisResultProvider);
    final useDesktopFrame = Platform.isWindows;
    final showCaptureAction = supportsRegionCapture;
    final showAppBar =
        !useDesktopFrame && MediaQuery.sizeOf(context).width >= 320;
    if (result == null) {
      final emptyState = Center(
        child: Wrap(
          spacing: 12,
          runSpacing: 12,
          alignment: WrapAlignment.center,
          children: [
            if (showCaptureAction)
              FilledButton.icon(
                onPressed: _captureBusy ? null : _captureAgain,
                icon: const Icon(Icons.screenshot_monitor_outlined),
                label: Text(_captureBusy ? '截屏中...' : '截屏分析'),
              ),
            OutlinedButton(
              onPressed: () => context.go('/'),
              child: const Text('返回首页导入图片'),
            ),
          ],
        ),
      );
      return Scaffold(
        appBar: showAppBar
            ? AppBar(
                leading: const _HomeBackButton(),
                title: const Text('分析结果'),
                actions: [
                  if (showCaptureAction)
                    _CaptureAgainIconButton(
                      busy: _captureBusy,
                      onPressed: _captureAgain,
                    ),
                ],
              )
            : null,
        body: useDesktopFrame
            ? DesktopPageFrame(
                title: '分析结果',
                currentRoute: '/result',
                leading: const _HomeBackButton(),
                actions: buildDesktopWindowActions(
                  context,
                  ref,
                  currentRoute: '/result',
                  trailing: [
                    if (showCaptureAction)
                      _CaptureAgainIconButton(
                        busy: _captureBusy,
                        onPressed: _captureAgain,
                      ),
                  ],
                ),
                child: emptyState,
              )
            : emptyState,
      );
    }

    final content = LayoutBuilder(
      builder: (context, constraints) {
        final hasPreview = result.sourceBytes.isNotEmpty;
        final wide = constraints.maxWidth >= 700;
        final results = _ResultPanel(result: result);
        final preview = _SourcePreviewCard(
          result: result,
          fillHeight: wide,
          showCaptureAction: showCaptureAction,
          captureBusy: _captureBusy,
          onCaptureAgain: _captureAgain,
        );

        if (!hasPreview) {
          if (!wide) {
            return ListView(
              padding: const EdgeInsets.all(20),
              children: [
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
              child: results,
            ),
          );
        }

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
    );

    return Scaffold(
      appBar: showAppBar
          ? AppBar(
              leading: const _HomeBackButton(),
              title: const Text('分析结果'),
              actions: [
                if (showCaptureAction)
                  _CaptureAgainIconButton(
                    busy: _captureBusy,
                    onPressed: _captureAgain,
                  ),
              ],
            )
          : null,
      body: useDesktopFrame
          ? DesktopPageFrame(
              title: '分析结果',
              currentRoute: '/result',
              leading: const _HomeBackButton(),
              actions: buildDesktopWindowActions(
                context,
                ref,
                currentRoute: '/result',
                trailing: [
                  if (showCaptureAction)
                    _CaptureAgainIconButton(
                      busy: _captureBusy,
                      onPressed: _captureAgain,
                    ),
                ],
              ),
              child: content,
            )
          : content,
    );
  }
}

class _HomeBackButton extends StatelessWidget {
  const _HomeBackButton();

  @override
  Widget build(BuildContext context) {
    return IconButton(
      tooltip: '返回首页',
      onPressed: () => context.go('/'),
      icon: const Icon(Icons.arrow_back_rounded),
    );
  }
}

class _SourcePreviewCard extends StatelessWidget {
  const _SourcePreviewCard({
    required this.result,
    required this.fillHeight,
    required this.showCaptureAction,
    required this.captureBusy,
    required this.onCaptureAgain,
  });

  final AnalysisResult result;
  final bool fillHeight;
  final bool showCaptureAction;
  final bool captureBusy;
  final VoidCallback onCaptureAgain;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: fillHeight ? MainAxisSize.max : MainAxisSize.min,
          children: [
            Wrap(
              spacing: 12,
              runSpacing: 12,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                const Text(
                  '图片预览',
                  style: TextStyle(fontWeight: FontWeight.w700),
                ),
                if (showCaptureAction)
                  OutlinedButton.icon(
                    onPressed: captureBusy ? null : onCaptureAgain,
                    icon: const Icon(Icons.screenshot_monitor_outlined),
                    label: Text(captureBusy ? '截屏中...' : '再次截屏'),
                  ),
              ],
            ),
            const SizedBox(height: 12),
            Text('来源：${result.sourceLabel}'),
            const SizedBox(height: 8),
            Text('分析时间：${result.analyzedAt.toLocal()}'),
            const SizedBox(height: 16),
            if (fillHeight)
              Expanded(child: _PreviewImage(result: result))
            else
              _PreviewImage(result: result),
          ],
        ),
      ),
    );
  }
}

class _CaptureAgainIconButton extends StatelessWidget {
  const _CaptureAgainIconButton({
    required this.busy,
    required this.onPressed,
  });

  final bool busy;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      tooltip: '再次截屏',
      onPressed: busy ? null : onPressed,
      icon: const Icon(Icons.screenshot_monitor_outlined),
    );
  }
}

class _PreviewImage extends StatelessWidget {
  const _PreviewImage({required this.result});

  final AnalysisResult result;

  @override
  Widget build(BuildContext context) {
    if (result.sourceBytes.isEmpty) {
      return ClipRRect(
        borderRadius: BorderRadius.circular(16),
        child: Container(
          width: double.infinity,
          constraints: const BoxConstraints(minHeight: 240, maxHeight: 360),
          color: const Color(0xFFF4F6FA),
          alignment: Alignment.center,
          padding: const EdgeInsets.all(24),
          child: const Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                Icons.edit_note_rounded,
                size: 40,
                color: Color(0xFF4F6DDC),
              ),
              SizedBox(height: 12),
              Text(
                '这是一次手动输入分析，没有原图预览。',
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      );
    }
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: Container(
        width: double.infinity,
        constraints: const BoxConstraints(minHeight: 240, maxHeight: 360),
        color: const Color(0xFFF4F6FA),
        alignment: Alignment.center,
        child: Image.memory(
          result.sourceBytes,
          fit: BoxFit.contain,
          errorBuilder: (_, __, ___) => const SizedBox.shrink(),
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
                  child: Text('未提取到有效的蛋尺寸/蛋重量。'),
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
    final title = entry.measurement.eggName?.trim().isNotEmpty == true
        ? '蛋 $index · ${entry.measurement.eggName!.trim()}'
        : '蛋 $index';
    return Card(
      elevation: 0,
      color: const Color(0xFFF7F9FD),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(height: 12),
            Wrap(
              spacing: 16,
              runSpacing: 8,
              children: [
                _MetricChip(
                  label: '蛋尺寸',
                  value:
                      '${entry.measurement.heightInMeters.toStringAsFixed(3)} m',
                ),
                _MetricChip(
                  label: '蛋重量',
                  value:
                      '${entry.measurement.weightInKg.toStringAsFixed(3)} kg',
                ),
              ],
            ),
            const SizedBox(height: 16),
            for (final indexed in entry.candidates.indexed) ...[
              _CandidateTile(
                candidate: indexed.$2,
                highlighted: indexed.$1 == 0,
              ),
              const SizedBox(height: 10),
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

class _CandidateTile extends StatelessWidget {
  const _CandidateTile({
    required this.candidate,
    required this.highlighted,
  });

  final Candidate candidate;
  final bool highlighted;

  @override
  Widget build(BuildContext context) {
    final borderColor =
        highlighted ? const Color(0xFF4F6DDC) : const Color(0xFFD8DEEC);
    final backgroundColor =
        highlighted ? const Color(0xFFEAF0FF) : Colors.white;
    final portraitSize = MediaQuery.sizeOf(context).width >= 420 ? 108.0 : 92.0;

    return DecoratedBox(
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
          color: borderColor,
          width: highlighted ? 2 : 1,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 12, 14, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (highlighted) ...[
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                decoration: BoxDecoration(
                  color: const Color(0xFF4F6DDC),
                  borderRadius: BorderRadius.circular(999),
                ),
                child: const Text(
                  '最高匹配',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 12,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              const SizedBox(height: 10),
            ],
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                PortraitImage(
                  portraitKey: candidate.portraitKey,
                  label: candidate.petName,
                  size: portraitSize,
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        '${candidate.petName} (#${candidate.petId})',
                        style: TextStyle(
                          fontWeight:
                              highlighted ? FontWeight.w800 : FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        '概率 ${candidate.probability.toStringAsFixed(2)}% | '
                        '蛋尺寸 ${candidate.heightRangeLabel} | '
                        '蛋重量 ${candidate.weightRangeLabel}\n'
                        '${candidate.matchLabel}'
                        '${candidate.hatchLabel == null ? '' : ' | 孵化 ${candidate.hatchLabel}'}',
                      ),
                    ],
                  ),
                ),
              ],
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
