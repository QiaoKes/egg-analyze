#!/usr/bin/env python3

import argparse
import os
import pathlib
import shutil
import subprocess
import sys
import venv


def main() -> int:
    parser = argparse.ArgumentParser(
        description='Create a bundled Python venv and install OCR requirements.',
    )
    parser.add_argument(
        'target',
        help='Target venv directory, such as .../Contents/Resources/python',
    )
    parser.add_argument(
        '--requirements',
        default='requirements-ocr.txt',
        help='Path to the pip requirements file.',
    )
    parser.add_argument(
        '--clear',
        action='store_true',
        help='Delete the target directory before creating the venv.',
    )
    parser.add_argument(
        '--system-site-packages',
        action='store_true',
        help='Allow the bundled venv to reuse packages from the base Python.',
    )
    args = parser.parse_args()

    target = pathlib.Path(args.target).resolve()
    requirements = pathlib.Path(args.requirements).resolve()

    if not requirements.is_file():
      raise SystemExit(f'Requirements file not found: {requirements}')

    if args.clear and target.exists():
        shutil.rmtree(target)

    target.parent.mkdir(parents=True, exist_ok=True)

    builder = venv.EnvBuilder(
        with_pip=True,
        clear=False,
        symlinks=True,
        system_site_packages=args.system_site_packages,
    )
    builder.create(str(target))

    python = bundled_python_executable(target)

    env = os.environ.copy()
    env.setdefault('PYTHONUTF8', '1')
    env.setdefault('PYTHONIOENCODING', 'utf-8')

    if not args.system_site_packages:
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
        return target / 'Scripts' / 'python.exe'
    return target / 'bin' / 'python3'


def run(command: list[str], *, env: dict[str, str]) -> None:
    subprocess.run(command, check=True, env=env)


if __name__ == '__main__':
    raise SystemExit(main())
