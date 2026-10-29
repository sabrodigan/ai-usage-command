#!/usr/bin/env python3
"""Install the CLI and plugin; preserve and back up the current bar layout."""
from pathlib import Path
import datetime
import json
import os
import shutil
import subprocess

source = Path(__file__).resolve().parent
config = Path(os.environ.get('XDG_CONFIG_HOME', str(Path.home() / '.config')))
plugin = config / 'omarchy/plugins/sabrodigan.todo'
shell = config / 'omarchy/shell.json'
layout = json.loads(shell.read_text())
backup = shell.with_name('shell.json.bak.todo.' + datetime.datetime.now().strftime('%Y%m%d%H%M%S%f'))
shutil.copy2(shell, backup)
plugin.mkdir(parents=True, exist_ok=True)
(plugin / 'bin').mkdir(exist_ok=True)
for name in ['Widget.qml', 'manifest.json', 'README.md', 'bin/todo']:
    shutil.copy2(source / name, plugin / name)
bin_dir = Path.home() / '.local/bin'
bin_dir.mkdir(parents=True, exist_ok=True)
shutil.copy2(source / 'bin/todo', bin_dir / 'todo')
sections = layout.setdefault('bar', {}).setdefault('layout', {})
if not any(item.get('id') == 'sabrodigan.todo' for items in sections.values() for item in items):
    sections.setdefault('left', []).append({'id': 'sabrodigan.todo'})
temporary = shell.with_suffix('.todo.tmp')
temporary.write_text(json.dumps(layout, indent=2) + '\n')
os.replace(temporary, shell)
version = json.loads((source / 'manifest.json').read_text())['version']
print(f'Installed todo CLI and bar plugin v{version}. Config backup: {backup}')
try:
    result = subprocess.run(['omarchy-shell', 'shell', 'rescanPlugins'],
                            capture_output=True, text=True, timeout=10)
    if result.returncode != 0:
        print('Could not request a live plugin reload: ' + (result.stderr.strip() or result.stdout.strip()))
    elif result.stdout.strip():
        print(result.stdout.strip())
except (OSError, subprocess.TimeoutExpired) as error:
    print(f'Could not request a live plugin reload: {error}')
print('If clicking the bar still shows the old list, run: omarchy restart shell')
print('To open the CRUD menu directly, run: todo interactive')
