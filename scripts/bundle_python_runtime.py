#!/usr/bin/env python3

import argparse
import os
import pathlib
import shutil
import subprocess
import sys


def main() -> int:
    parser = argparse.ArgumentParser(
        description='Copy a full Python runtime into the app package and install OCR requirements.',
    )
    parser.add_argument('source', help='Source Python runtime directory.')
    parser.add_argument('target', help='Target runtime directory inside the packaged app.')
    parser.add_argument(
        '--requirements',
        default='requirements-ocr.txt',
        help='Path to the pip requirements file.',
    )
    parser.add_argument(
        '--clear',
        action='store_true',
        help='Delete the target directory before copying.',
    )
    args = parser.parse_args()

    source = pathlib.Path(args.source).resolve()
    target = pathlib.Path(args.target).resolve()
    requirements = pathlib.Path(args.requirements).resolve()

    if not source.is_dir():
        raise SystemExit(f'Python runtime not found: {source}')
    if not requirements.is_file():
        raise SystemExit(f'Requirements file not found: {requirements}')

    if args.clear and target.exists():
        shutil.rmtree(target)

    if target.exists():
        raise SystemExit(f'Target already exists: {target}')

    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(source, target, symlinks=True)

    python = bundled_python_executable(target)
    env = os.environ.copy()
    env.setdefault('PYTHONUTF8', '1')
    env.setdefault('PYTHONIOENCODING', 'utf-8')

    run([str(python), '-m', 'pip', 'install', '--upgrade', 'pip'], env=env)
    run(
        [
            str(python),
            '-m',
            'pip',
            'install',
            '--prefer-binary',
            '--no-compile',
            '-r',
            str(requirements),
        ],
        env=env,
    )
    return 0


def bundled_python_executable(target: pathlib.Path) -> pathlib.Path:
    if sys.platform.startswith('win'):
        return target / 'python.exe'
    return target / 'bin' / 'python3'


def run(command: list[str], *, env: dict[str, str]) -> None:
    subprocess.run(command, check=True, env=env)


if __name__ == '__main__':
    raise SystemExit(main())
