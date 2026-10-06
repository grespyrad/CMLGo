#!/usr/bin/env python3
"""Проверка нейтральности собственных примеров, тестов и документации."""
import argparse
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXCLUDED = ('testdata/upstream/', 'testdata/conformance/', 'internal/grammar/',
            'bench/grammar/', 'bench/cpp/grammar.inc', 'scripts/check-examples.py')
PATTERNS = (
    re.compile(rb'\b(?:Home|House|Casa|Ventilation|Fan|HomeComfort|Toilet|MotorizedWindow|HomeAssistant(?:Integration)?)\b'),
    re.compile(rb'home\.cml|ValidateHome|internal/domain/ventilation'),
    re.compile(rb'/(?:Users|home)/[A-Za-z0-9_.-]+/'),
    re.compile(rb'[A-Za-z0-9._%+-]+@(?:gmail\.com|mail\.ru|yandex\.(?:ru|com))'),
    re.compile(rb'gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----'),
)


def violation(filename, contents):
    """Upstream attribution отделена от собственных материалов."""
    if filename.startswith(EXCLUDED):
        return False
    if re.search(r'\b(?:Дом|Вентиляция|Вентилятор)\b', contents.decode('utf-8', errors='replace')):
        return True
    return filename.endswith(('.cpu', '.mem')) or any(pattern.search(contents) for pattern in PATTERNS)


def current():
    """Проверяет tracked и non-ignored untracked файлы до commit."""
    names = subprocess.check_output(
        ['git', 'ls-files', '-co', '--exclude-standard'], cwd=ROOT, text=True).splitlines()
    return sorted({name for name in names if (ROOT / name).is_file()
                   and violation(name, (ROOT / name).read_bytes())})


def history():
    """Проверяет файлы всех достижимых веток и тегов, без вывода содержимого."""
    lines = subprocess.check_output(['git', 'rev-list', '--objects', '--all'], cwd=ROOT).splitlines()
    paths = {line.split(b' ', 1)[0]: line.split(b' ', 1)[1].decode() if b' ' in line else ''
             for line in lines}
    data = subprocess.check_output(['git', 'cat-file', '--batch'], cwd=ROOT,
                                   input=b'\n'.join(paths) + b'\n')
    position = 0
    findings = set()
    while position < len(data):
        end = data.index(b'\n', position)
        identifier, kind, size = data[position:end].split()
        size = int(size)
        contents = data[end + 1:end + 1 + size]
        position = end + size + 2
        if kind == b'blob' and violation(paths[identifier], contents):
            findings.add(paths[identifier])
    return sorted(findings)


def self_test():
    """Отрицательные пробы доказывают отказ при повторном переносе личного контекста."""
    assert violation('example_test.go', b'BoundedContext Home')
    assert violation('testdata/reference/model.cml', b'Entity Fan')
    assert violation('README.md', ('/' + 'Users' + '/sample/project/file').encode())
    assert violation('bench/results/profile.cpu', b'compressed-data')
    assert not violation('testdata/upstream/model.cml', b'BoundedContext Home')
    assert not violation('example_test.go', 'Сервис 文 🔧'.encode())
    assert not violation('examples/model.cml', b'BoundedContext ServiceIntegration')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--history', action='store_true')
    parser.add_argument('--self-test', action='store_true')
    args = parser.parse_args()
    if args.self_test:
        self_test()
        print('Neutral example policy self-test passed.')
    else:
        findings = history() if args.history else current()
        for name in findings:
            print('Non-neutral material:', name)
        print('Neutral example policy:', len(findings), 'violations.')
        raise SystemExit(bool(findings))
