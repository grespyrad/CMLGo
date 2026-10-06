#!/usr/bin/env python3
"""Check the library/tool boundary and prove rejection with negative fixtures."""
import argparse,re,tempfile
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--mode',choices=['library','cli'],required=True);p.add_argument('--self-test',action='store_true');args=p.parse_args()
root=Path(__file__).resolve().parents[1]
def inspect(root,mode):
 errors=[]
 for file in root.rglob('*.go'):
  relative=file.relative_to(root)
  if any(part in ['.git','.agents','node_modules','tools','scripts','bench','bin'] for part in relative.parts):continue
  source=file.read_text()
  if mode=='library' and relative.parts[0]=='cmd':errors.append(str(relative)+' library contains CLI')
  if mode=='cli' and file.name in ['parser.go','models_generated.go','grammar.go']:errors.append(str(relative)+' copies library implementation')
  if mode=='library' and 'func init(' in source:errors.append(str(relative)+' init side effect')
  for match in re.finditer(r'^import\s*(?:\((.*?)\)|([^\n]+))',source,re.S|re.M):
   imports=re.findall(r'"([^"]+)"',match.group(1) or match.group(2))
   if mode=='library':
    for package in imports:
     if package=='os/exec' or package.startswith('net/') and package!='net/url' or package=='net':errors.append(str(relative)+' forbidden runtime import '+package)
     if '.' in package.split('/')[0] and not package.startswith('github.com/grespyrad/CMLGo'):errors.append(str(relative)+' external runtime dependency '+package)
 if mode=='cli':
  module=(root/'go.mod').read_text()
  if not re.search(r'github.com/grespyrad/CMLGo\s+v\d+\.\d+\.\d+',module):errors.append('CMLGo is not pinned')
  if 'replace ' in module:errors.append('local replace in released module')
 return errors
if args.self_test:
 with tempfile.TemporaryDirectory() as directory:
  fixture=Path(directory);(fixture/'go.mod').write_text('module fixture\n')
  (fixture/'bad.go').write_text('package library\nimport "os/exec"\nfunc init() {}\n')
  if len(inspect(fixture,'library'))<2:raise SystemExit('architecture checker failed negative library probe')
  (fixture/'bad.go').unlink();(fixture/'parser.go').write_text('package copied\n')
  if len(inspect(fixture,'cli'))<2:raise SystemExit('architecture checker failed negative CLI probe')
else:
 errors=inspect(root,args.mode)
 if errors:raise SystemExit('\n'.join(errors))
