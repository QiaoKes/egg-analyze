import 'package:egg_ocr/egg_ocr.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../shared/desktop_window_actions.dart';
import '../../../shared/providers.dart';

class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({super.key});

  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends ConsumerState<SettingsPage> {
  bool _refreshing = false;
  bool _clearing = false;

  @override
  Widget build(BuildContext context) {
    final dataset = ref.watch(datasetSnapshotProvider);
    final ocrLabel = currentOcrEngineLabel();

    return Scaffold(
      appBar: AppBar(
        title: const Text('设置'),
        actions: buildDesktopWindowActions(
          context,
          ref,
          currentRoute: '/settings',
        ),
      ),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Card(
            child: ListTile(
              title: const Text('版本'),
              subtitle: const Text(appVersion),
            ),
          ),
          const SizedBox(height: 16),
          Card(
            child: ListTile(
              title: const Text('当前 OCR 引擎'),
              subtitle: Text(ocrLabel),
            ),
          ),
          const SizedBox(height: 16),
          Card(
            child: ListTile(
              title: const Text('OCR 基准测试'),
              subtitle: const Text('用内置样本图真实调用 OCR 并统计提取命中率'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => context.push('/benchmark'),
            ),
          ),
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('数据刷新',
                      style: TextStyle(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 12),
                  dataset.when(
                    data: (snapshot) => Text(
                      '最近更新时间：${snapshot.updatedAt.toLocal()}\n'
                      '来源：${snapshot.fromCache ? '本地缓存' : '远程拉取'}',
                    ),
                    error: (error, _) => Text('数据状态异常：$error'),
                    loading: () => const CircularProgressIndicator.adaptive(),
                  ),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    onPressed: _refreshing ? null : _refreshDataset,
                    icon: const Icon(Icons.refresh),
                    label: Text(_refreshing ? '刷新中...' : '手动刷新数据'),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('图片缓存',
                      style: TextStyle(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    onPressed: _clearing ? null : _clearPortraitCache,
                    icon: const Icon(Icons.image_not_supported_outlined),
                    label: const Text('清理图片缓存'),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _refreshDataset() async {
    setState(() => _refreshing = true);
    try {
      await ref.read(datasetRepositoryProvider).refresh();
      ref.invalidate(datasetSnapshotProvider);
    } finally {
      if (mounted) {
        setState(() => _refreshing = false);
      }
    }
  }

  Future<void> _clearPortraitCache() async {
    setState(() => _clearing = true);
    try {
      await ref.read(portraitRepositoryProvider).clearCache();
    } finally {
      if (mounted) {
        setState(() => _clearing = false);
      }
    }
  }
}
