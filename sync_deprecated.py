#!/usr/bin/env python3
"""
sync_deprecated.py — 从 Unciv.jar 自动提取废弃 Unique 规则，生成 Go 代码。
用法: python3 sync_deprecated.py [Unciv.jar路径] [输出.go路径]
每次 Unciv 更新后运行一次即可。
"""

import zipfile, subprocess, os, re, textwrap, tempfile, shutil, sys

def extract_unique_patterns(jar_path: str) -> list[str]:
    tmp = tempfile.mkdtemp()
    with zipfile.ZipFile(jar_path) as zf:
        for f in zf.namelist():
            if 'DeprecatedUniqueType' in f and f.endswith('.class'):
                zf.extract(f, tmp)
    cp = os.path.join(tmp, 'com', 'unciv', 'models', 'ruleset', 'unique', 'DeprecatedUniqueType.class')
    result = subprocess.run(['javap', '-verbose', '-p', cp], capture_output=True, text=True)

    utf8s = {}
    for line in result.stdout.split('\n'):
        m = re.match(r'#(\d+)\s*=\s*Utf8\s+(.*)', line.strip())
        if m:
            utf8s[int(m.group(1))] = m.group(2).strip()

    keywords = [
        'Gold', 'Strength', 'Happiness', 'Culture', 'Science', 'Production',
        'Food', 'Faith', 'Movement', 'Experience', 'Damage', 'Heal', 'Promotion',
        'Specialist', 'Population', 'Maintenance', 'Upgrade', 'Paradrop',
        'Religion', 'Espionage', 'Victory', 'Policy', 'Defense', 'Attack',
        'Construct', 'Improve', 'Convert', 'Discover', 'Spread', 'Annex', 'Puppet',
        'Embark', 'Withdraw', 'Stack', 'Garrison', 'Luxury', 'Strategic',
        'Encampment', 'Barbarian', 'Merchant', 'Prophet', 'Pantheon',
        'Golden Age', 'Wonder', 'Trade Route', 'Unit', 'Building', 'Tile',
        'Resource', 'City', 'City-State', 'Empire', 'Technology',
        'Free', 'Gain', 'Receive', 'Grants', 'Provides', 'Double', 'Triple',
        'Hidden', 'Requires', 'Cannot', 'Only available', 'Not displayed',
        'Unavailable', 'May buy', 'May Paradrop', 'Can construct', 'Can spend',
        'Can only', 'Enables', 'Incompatible', 'Unlocked', 'Costs',
        'Consuming', 'Bonus', 'Extra', 'Remove', 'Lose',
        'Starts', 'Ending', 'Upon ', 'After ', 'Before ',
        'Quantity', 'adjacent', 'neighbor', 'embarked', 'coast',
        'ocean', 'Forest', 'Jungle', 'Snow', 'Tundra', 'Hill',
        'road', 'railroad', 'nuclear', 'nuke',
        'Yield', 'percent', 'carried over', 'border growth',
        'stat', 'stats', 'amount', 'turns', 'XP', 'HP',
        'Great Person', 'Great Prophet', 'Melee', 'Ranged', 'Land', 'Water',
        'All units', 'mapUnitFilter', 'baseUnitFilter',
        'tileFilter', 'cityFilter', 'combatantFilter',
        'buildingFilter', 'improvementName', 'buildingName',
        'relativeAmount', 'positiveAmount', 'simpleTerrain',
        'Followers', 'Majority Religion',
        'Civilization', 'Capital',
        'defend', 'attacking', 'fighting', 'pillage', 'plunder', 'conquer',
        'Garrison', 'Stacked', 'Wounded', 'Embarked', 'Embarkation',
        'Annexed', 'Puppeted', 'Pantheon',
    ]

    def is_pattern(s):
        if len(s) < 15 or len(s) > 200:
            return False
        if 'Lcom/' in s or 'Ljava/' in s:
            return False
        # CamelCase enum names
        if ' ' not in s and s[0].isupper() and '[' not in s:
            return False
        # Pure noise
        noise = ['checkNotNull', 'expression', 'value', 'parameter', 'constructor']
        if s.lower() in noise:
            return False
        return any(kw.lower() in s.lower() for kw in keywords)

    patterns, seen = [], set()
    for idx in sorted(utf8s):
        s = utf8s[idx]
        if is_pattern(s) and s not in seen:
            seen.add(s)
            patterns.append(s)
    shutil.rmtree(tmp, ignore_errors=True)
    return patterns


def generate_go(patterns: list[str]) -> str:
    header = textwrap.dedent('''\
    package app

    // ⚠️ 由 sync_deprecated.py 自动生成 — 来源: Unciv.jar DeprecatedUniqueType
    // 每次更新 Unciv 后运行: python3 sync_deprecated.py

    var deprecatedRulesAuto = []deprecatedRule{
    ''')
    body = []
    for p in patterns:
        escaped = p.replace('\\', '\\\\').replace('"', '\\"')
        severity = "error" if any(kw in p.lower() for kw in
            ['obsolete', 'removed', 'as of 3.', 'extremely old']) else "warning"
        body.append(f'\t{{Since: "auto", Severity: "{severity}", Pattern: "{escaped}"}},\n')
    footer = '}\n'
    return header + ''.join(body) + footer


if __name__ == '__main__':
    jar = sys.argv[1] if len(sys.argv) > 1 else r'C:\Users\drj13\Desktop\官方unciv文件\Unciv.jar'
    out = sys.argv[2] if len(sys.argv) > 2 else r'C:\Users\drj13\Desktop\unciv-mod-manager\internal\app\deprecated_gen.go'
    patterns = extract_unique_patterns(jar)
    print(f"提取到 {len(patterns)} 条废弃 unique 模式")
    with open(out, 'w', encoding='utf-8') as f:
        f.write(generate_go(patterns))
    print(f"已写入 {out}")
