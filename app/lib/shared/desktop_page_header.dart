import 'dart:io';

import 'package:flutter/material.dart';
import 'package:window_manager/window_manager.dart';

class DesktopPageFrame extends StatelessWidget {
  const DesktopPageFrame({
    super.key,
    required this.title,
    required this.child,
    this.leading,
    this.actions = const [],
  });

  final String title;
  final Widget child;
  final Widget? leading;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return DragToResizeArea(
      resizeEdgeSize: 6,
      child: ColoredBox(
        color: theme.scaffoldBackgroundColor,
        child: Column(
          children: [
            DesktopPageHeader(
              title: title,
              leading: leading,
              actions: actions,
            ),
            Expanded(child: child),
          ],
        ),
      ),
    );
  }
}

class DesktopPageHeader extends StatelessWidget {
  const DesktopPageHeader({
    super.key,
    required this.title,
    this.leading,
    this.actions = const [],
  });

  final String title;
  final Widget? leading;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final dividerColor = theme.dividerColor.withValues(alpha: 0.55);
    final titleStyle = theme.textTheme.titleMedium?.copyWith(
      fontWeight: FontWeight.w700,
    );

    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final showLeading = leading != null && width >= 120;
        final showActions = actions.isNotEmpty && width >= 220;
        final showTitle = width >= 96;
        final trailing = <Widget>[
          if (showActions) ...actions,
          if (Platform.isWindows) ...[
            if (showActions) const SizedBox(width: 8),
            _DesktopWindowControls(brightness: theme.brightness),
          ],
        ];

        return DecoratedBox(
          decoration: BoxDecoration(
            color: theme.scaffoldBackgroundColor,
            border: Border(bottom: BorderSide(color: dividerColor)),
          ),
          child: SizedBox(
            height: 48,
            child: Row(
              children: [
                if (showLeading) ...[
                  const SizedBox(width: 8),
                  leading!,
                ] else
                  const SizedBox(width: 12),
                Expanded(
                  child: DragToMoveArea(
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 8),
                        child: Text(
                          showTitle ? title : '',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: titleStyle,
                        ),
                      ),
                    ),
                  ),
                ),
                if (trailing.isNotEmpty)
                  Flexible(
                    child: Align(
                      alignment: Alignment.centerRight,
                      child: Padding(
                        padding: const EdgeInsets.only(left: 8),
                        child: FittedBox(
                          fit: BoxFit.scaleDown,
                          alignment: Alignment.centerRight,
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: trailing,
                          ),
                        ),
                      ),
                    ),
                  )
                else
                  const SizedBox(width: 12),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _DesktopWindowControls extends StatefulWidget {
  const _DesktopWindowControls({required this.brightness});

  final Brightness brightness;

  @override
  State<_DesktopWindowControls> createState() => _DesktopWindowControlsState();
}

class _DesktopWindowControlsState extends State<_DesktopWindowControls>
    with WindowListener {
  bool _isMaximized = false;

  @override
  void initState() {
    super.initState();
    windowManager.addListener(this);
    _syncMaximized();
  }

  @override
  void dispose() {
    windowManager.removeListener(this);
    super.dispose();
  }

  Future<void> _syncMaximized() async {
    final isMaximized = await windowManager.isMaximized();
    if (!mounted) {
      return;
    }
    setState(() => _isMaximized = isMaximized);
  }

  @override
  void onWindowMaximize() {
    if (mounted) {
      setState(() => _isMaximized = true);
    }
  }

  @override
  void onWindowUnmaximize() {
    if (mounted) {
      setState(() => _isMaximized = false);
    }
  }

  @override
  void onWindowRestore() {
    _syncMaximized();
  }

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        WindowCaptionButton.minimize(
          brightness: widget.brightness,
          onPressed: () => windowManager.minimize(),
        ),
        if (_isMaximized)
          WindowCaptionButton.unmaximize(
            brightness: widget.brightness,
            onPressed: () => windowManager.unmaximize(),
          )
        else
          WindowCaptionButton.maximize(
            brightness: widget.brightness,
            onPressed: () => windowManager.maximize(),
          ),
        WindowCaptionButton.close(
          brightness: widget.brightness,
          onPressed: () => windowManager.close(),
        ),
      ],
    );
  }
}
