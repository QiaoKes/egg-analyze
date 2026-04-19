import 'dart:io';

import 'package:desktop_drop/desktop_drop.dart';
import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
import 'package:screen_capturer/screen_capturer.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:window_manager/window_manager.dart';

import '../../../shared/desktop_window_actions.dart';
import '../../../shared/desktop_page_header.dart';
import '../../../shared/providers.dart';

class HomePage extends ConsumerStatefulWidget {
  const HomePage({super.key});

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

class _HomePageState extends ConsumerState<HomePage> {
  static final Uri _projectUri =
      Uri.parse('https://github.com/QiaoKes/egg-analyze');

  bool _dragging = false;
  bool _busy = false;

  Future<void> _analyzeXFile(XFile file) async {
    setState(() => _busy = true);
    try {
      final bytes = await file.readAsBytes();
      await ref.read(analysisControllerProvider).analyzeBytes(
            bytes,
            label: file.name,
          );
      if (mounted) {
        context.go('/result');
      }
    } catch (error) {
      if (!mounted) {
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('导入失败：$error')),
      );
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _pickImage(ImageSource source) async {
    final isDesktop = Platform.isMacOS || Platform.isWindows;
    if (isDesktop && source == ImageSource.gallery) {
      const typeGroup = XTypeGroup(
        label: 'images',
        extensions: <String>['png', 'jpg', 'jpeg', 'webp'],
      );
      final file = await openFile(acceptedTypeGroups: <XTypeGroup>[typeGroup]);
      if (file == null) {
        return;
      }
      await _analyzeXFile(file);
      return;
    }

    final picker = ImagePicker();
    final file = await picker.pickImage(source: source);
    if (file == null) {
      return;
    }
    await _analyzeXFile(file);
  }

  Future<void> _captureRegionAndAnalyze() async {
    if (_busy) {
      return;
    }
    setState(() => _busy = true);
    final shouldHideWindow = Platform.isMacOS || Platform.isWindows;
    try {
      if (Platform.isMacOS) {
        final allowed = await screenCapturer.isAccessAllowed();
        if (!allowed) {
          await screenCapturer.requestAccess();
          if (!mounted) {
            return;
          }
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('请先在系统设置中允许屏幕录制权限，然后再次点击截屏分析。'),
            ),
          );
          return;
        }
      }

      if (shouldHideWindow) {
        await windowManager.hide();
        await Future<void>.delayed(const Duration(milliseconds: 160));
      }

      final captured = await screenCapturer.capture(
        mode: CaptureMode.region,
        imagePath: null,
        copyToClipboard: true,
        silent: true,
      );
      if (captured?.imageBytes == null) {
        return;
      }

      await ref.read(analysisControllerProvider).analyzeBytes(
            captured!.imageBytes!,
            label: '区域截图',
          );
      if (mounted && (Platform.isMacOS || Platform.isWindows)) {
        final controller = ref.read(desktopWindowControllerProvider);
        await controller.transitionToBubble(() {
          if (context.mounted) {
            context.go('/bubble');
          }
        });
      } else if (mounted) {
        context.go('/result');
      }
    } catch (error) {
      if (!mounted) {
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('截屏分析失败：$error')),
      );
    } finally {
      if (shouldHideWindow) {
        await windowManager.show();
        await windowManager.focus();
      }
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _openProjectLink() async {
    final launched = await launchUrl(
      _projectUri,
      mode: LaunchMode.externalApplication,
    );
    if (!launched && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('项目地址打开失败')),
      );
    }
  }

  Future<void> _openManualInputDialog() async {
    if (_busy) {
      return;
    }
    final input = await showDialog<_ManualMeasurementInput>(
      context: context,
      builder: (dialogContext) => const _ManualInputDialog(),
    );
    if (input == null || !mounted) {
      return;
    }

    setState(() => _busy = true);
    try {
      await ref.read(analysisControllerProvider).analyzeManualMeasurement(
            heightInMeters: input.heightInMeters,
            weightInKg: input.weightInKg,
          );
      if (mounted) {
        context.go('/result');
      }
    } catch (error) {
      if (!mounted) {
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('手动分析失败：$error')),
      );
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final dataset = ref.watch(datasetSnapshotProvider);
    final isDesktop = Platform.isMacOS || Platform.isWindows;
    final useDesktopFrame = Platform.isWindows;
    final showAppBar =
        !useDesktopFrame && MediaQuery.sizeOf(context).width >= 320;

    Widget importer = _ImportCard(
      busy: _busy,
      dragging: _dragging,
      showScreenshotAction: isDesktop,
      onOpenImage: () => _pickImage(ImageSource.gallery),
      onManualInput: _openManualInputDialog,
      onCaptureScreenshot: isDesktop ? _captureRegionAndAnalyze : null,
      onTakePhoto:
          Platform.isAndroid ? () => _pickImage(ImageSource.camera) : null,
    );

    if (isDesktop) {
      importer = DropTarget(
        onDragEntered: (_) => setState(() => _dragging = true),
        onDragExited: (_) => setState(() => _dragging = false),
        onDragDone: (detail) async {
          setState(() => _dragging = false);
          if (detail.files.isEmpty) {
            return;
          }
          await _analyzeXFile(detail.files.first);
        },
        child: importer,
      );
    }

    final body = SafeArea(
      child: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          importer,
          const SizedBox(height: 20),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: dataset.when(
                data: (snapshot) => Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('数据状态',
                        style: TextStyle(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    Text(snapshot.fromCache ? '来源：本地缓存' : '来源：远程拉取'),
                    Text('更新时间：${snapshot.updatedAt.toLocal()}'),
                    Text('精灵数量：${snapshot.dataset.pets.length}'),
                  ],
                ),
                error: (error, _) => Text('数据加载失败：$error'),
                loading: () => const SizedBox(
                  height: 64,
                  child: Center(child: CircularProgressIndicator.adaptive()),
                ),
              ),
            ),
          ),
          const SizedBox(height: 20),
          Card(
            child: InkWell(
              borderRadius: BorderRadius.circular(12),
              onTap: _openProjectLink,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    const Icon(Icons.link_rounded),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text(
                            '项目地址',
                            style: TextStyle(fontWeight: FontWeight.w600),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            _projectUri.toString(),
                            style: TextStyle(
                              color: Theme.of(context).colorScheme.primary,
                              decoration: TextDecoration.underline,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 12),
                    const Icon(Icons.open_in_new_rounded),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );

    return Scaffold(
      appBar: showAppBar
          ? AppBar(
              title: const Text('洛克王国精灵蛋分析'),
              actions: [
                ...buildDesktopWindowActions(
                  context,
                  ref,
                  currentRoute: '/',
                ),
                IconButton(
                  tooltip: '设置',
                  onPressed: () => context.push('/settings'),
                  icon: const Icon(Icons.settings_outlined),
                ),
              ],
            )
          : null,
      body: useDesktopFrame
          ? DesktopPageFrame(
              title: '洛克王国精灵蛋分析',
              currentRoute: '/',
              actions: [
                ...buildDesktopWindowActions(
                  context,
                  ref,
                  currentRoute: '/',
                ),
                IconButton(
                  tooltip: '设置',
                  onPressed: () => context.push('/settings'),
                  icon: const Icon(Icons.settings_outlined),
                ),
              ],
              child: body,
            )
          : body,
    );
  }
}

class _ImportCard extends StatelessWidget {
  const _ImportCard({
    required this.busy,
    required this.dragging,
    required this.showScreenshotAction,
    required this.onOpenImage,
    required this.onManualInput,
    required this.onCaptureScreenshot,
    this.onTakePhoto,
  });

  final bool busy;
  final bool dragging;
  final bool showScreenshotAction;
  final VoidCallback onOpenImage;
  final VoidCallback onManualInput;
  final VoidCallback? onCaptureScreenshot;
  final VoidCallback? onTakePhoto;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: dragging ? const Color(0xFFE3ECFF) : null,
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('开始分析',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
            const SizedBox(height: 8),
            const Text('可以通过图片、手动输入数值，或截取当前画面来分析精灵蛋。'),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                FilledButton.icon(
                  onPressed: busy ? null : onOpenImage,
                  icon: const Icon(Icons.photo_library_outlined),
                  label: Text(busy ? '分析中...' : '打开图片'),
                ),
                OutlinedButton.icon(
                  onPressed: busy ? null : onManualInput,
                  icon: const Icon(Icons.edit_note_rounded),
                  label: Text(busy ? '分析中...' : '手动输入'),
                ),
                if (showScreenshotAction)
                  OutlinedButton.icon(
                    onPressed: busy ? null : onCaptureScreenshot,
                    icon: const Icon(Icons.screenshot_monitor_outlined),
                    label: Text(busy ? '分析中...' : '截屏分析'),
                  ),
                if (onTakePhoto != null)
                  OutlinedButton.icon(
                    onPressed: busy ? null : onTakePhoto,
                    icon: const Icon(Icons.photo_camera_outlined),
                    label: const Text('拍照'),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _ManualInputDialog extends StatefulWidget {
  const _ManualInputDialog();

  @override
  State<_ManualInputDialog> createState() => _ManualInputDialogState();
}

class _ManualInputDialogState extends State<_ManualInputDialog> {
  final _formKey = GlobalKey<FormState>();
  final _heightController = TextEditingController();
  final _weightController = TextEditingController();

  @override
  void dispose() {
    _heightController.dispose();
    _weightController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('手动输入精灵蛋数据'),
      content: Form(
        key: _formKey,
        child: SizedBox(
          width: 360,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(
                controller: _heightController,
                keyboardType: const TextInputType.numberWithOptions(
                  decimal: true,
                ),
                decoration: const InputDecoration(
                  labelText: '蛋尺寸',
                  hintText: '例如 0.810',
                  suffixText: 'm',
                ),
                validator: (value) => _validateMeasurement(value, label: '蛋尺寸'),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _weightController,
                keyboardType: const TextInputType.numberWithOptions(
                  decimal: true,
                ),
                decoration: const InputDecoration(
                  labelText: '蛋重量',
                  hintText: '例如 19.800',
                  suffixText: 'kg',
                ),
                validator: (value) => _validateMeasurement(value, label: '蛋重量'),
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('取消'),
        ),
        FilledButton(
          onPressed: _submit,
          child: const Text('开始分析'),
        ),
      ],
    );
  }

  String? _validateMeasurement(String? value, {required String label}) {
    final normalized = value?.trim() ?? '';
    if (normalized.isEmpty) {
      return '请输入$label';
    }
    final parsed = double.tryParse(normalized);
    if (parsed == null) {
      return '$label格式不正确';
    }
    if (parsed <= 0) {
      return '$label必须大于 0';
    }
    return null;
  }

  void _submit() {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    Navigator.of(context).pop(
      _ManualMeasurementInput(
        heightInMeters: double.parse(_heightController.text.trim()),
        weightInKg: double.parse(_weightController.text.trim()),
      ),
    );
  }
}

class _ManualMeasurementInput {
  const _ManualMeasurementInput({
    required this.heightInMeters,
    required this.weightInKg,
  });

  final double heightInMeters;
  final double weightInKg;
}
