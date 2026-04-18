import 'dart:io';

import 'package:desktop_drop/desktop_drop.dart';
import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:screen_capturer/screen_capturer.dart';
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

      final tempDir = await getTemporaryDirectory();
      final imagePath =
          '${tempDir.path}/egg-analyze-v2-screenshot-${DateTime.now().millisecondsSinceEpoch}.png';

      final captured = await screenCapturer.capture(
        mode: CaptureMode.region,
        imagePath: imagePath,
        copyToClipboard: false,
        silent: true,
      );
      if (captured?.imageBytes == null) {
        return;
      }

      await ref.read(analysisControllerProvider).analyzeBytes(
            captured!.imageBytes!,
            label: '区域截图',
          );
      if (mounted) {
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
        ],
      ),
    );

    return Scaffold(
      appBar: showAppBar
          ? AppBar(
              title: const Text('Egg Analyze v2'),
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
              title: 'Egg Analyze v2',
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
    required this.onCaptureScreenshot,
    this.onTakePhoto,
  });

  final bool busy;
  final bool dragging;
  final bool showScreenshotAction;
  final VoidCallback onOpenImage;
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
            const Text('导入图片',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
            const SizedBox(height: 8),
            const Text('桌面端支持打开图片和拖拽导入，Android 支持相册、拍照与系统分享。'),
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
