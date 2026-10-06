#!/usr/bin/env python3
"""Compile the pinned Xtext grammar subset; no runtime Python dependency."""
import ast
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
TOKEN = re.compile(r'''/\*.*?\*/|//[^\n]*|'(?:\\.|[^'\\])*'|"(?:\\.|[^"\\])*"|[A-Za-z_][A-Za-z_0-9]*|\+=|\?=|::|[^\s]''', re.S)

class Parser:
    def __init__(self, tokens):
        self.tokens, self.i = tokens, 0

    def peek(self):
        return self.tokens[self.i] if self.i < len(self.tokens) else ""

    def pop(self, expected=None):
        value = self.peek()
        if expected is not None and value != expected:
            raise ValueError((expected, value, self.tokens[max(0, self.i-5):self.i+5]))
        self.i += 1
        return value

    def group(self, op, sub):
        values = [sub()]
        while self.peek() == op:
            self.pop()
            values.append(sub())
        return values[0] if len(values) == 1 else {"op": {"|":"alt", "&":"unordered"}[op], "args": values}

    def expr(self):
        return self.group("|", lambda: self.group("&", self.sequence))

    def sequence(self):
        values = []
        while self.peek() and self.peek() not in ("|", "&", ")", ";"):
            values.append(self.item())
        return values[0] if len(values) == 1 else {"op":"seq", "args":values}

    def item(self):
        if self.i+1 < len(self.tokens) and self.tokens[self.i+1] in ("=", "+=", "?="):
            field, mode = self.pop(), self.pop()
            result = {"op":"assign", "text":field, "mode":mode, "args":[self.atom()]}
        else:
            result = self.atom()
        if self.peek() in ("?", "*", "+"):
            result = {"op":"repeat", "text":self.pop(), "args":[result]}
        return result

    def atom(self):
        token = self.pop()
        if token == "(":
            result = self.expr()
            self.pop(")")
            return result
        if token == "[":
            target = self.pop()
            if self.peek() == "::":
                self.pop()
                target = self.pop()
            if self.peek() == "|":
                raise ValueError("Unexpected explicit cross-reference syntax")
            self.pop("]")
            return {"op":"ref", "text":target}
        if token == "{":
            target = self.pop()
            self.pop("}")
            return {"op":"action", "text":target}
        if token.startswith(("'", '"')):
            return {"op":"lit", "text":ast.literal_eval(token)}
        if not re.fullmatch(r"[A-Za-z_][A-Za-z_0-9]*", token):
            raise ValueError(("Unsupported grammar token", token))
        return {"op":"call", "text":token}

rules = {}
for filename in ("TacticDDDLanguage.xtext", "ContextMappingDSL.xtext"):
    tokens = [t for t in TOKEN.findall((ROOT/"internal/grammar"/filename).read_text()) if not t.startswith(("/*", "//"))]
    p = Parser(tokens)
    while p.i < len(tokens):
        kind = "rule"
        if p.peek() in ("terminal", "enum"):
            kind = p.pop()
        if p.i+1 < len(tokens) and tokens[p.i+1] == ":":
            name = p.pop()
            p.pop(":")
            expr = p.expr()
            p.pop(";")
            rules[name] = {"kind":kind, "expr":expr}
        else:
            p.pop()

# Datatype parser rules return their matched text, not model nodes.
datatypes = {"Type", "JavaIdentifier", "ChannelIdentifier", "ThrowsIdentifier", "UserActivityDefaultVerb"}
for name in datatypes:
    rules[name]["kind"] = "datatype"

def normalize(expr, kind):
    if kind == "enum" and expr["op"] == "assign":
        return normalize(expr["args"][0], kind)
    if kind == "enum" and expr["op"] == "call":
        return {"op":"lit", "text":expr["text"]}
    return {**expr, **({"args":[normalize(x, kind) for x in expr["args"]]} if "args" in expr else {})}

for rule in rules.values():
    rule["expr"] = normalize(rule["expr"], rule["kind"])
# Исходная грамматика хранится отдельно для strict original режима.
(ROOT/"internal/grammar/original.json").write_text(json.dumps(rules, ensure_ascii=False, indent=2)+"\n")
base = rules.copy()
from copy import deepcopy
rules = deepcopy(rules)
# Overlay расширения не изменяет upstream .xtext.
invariant_tokens = TOKEN.findall((ROOT/"internal/grammar/Invariants.xtext").read_text())
invariant_tokens = [t for t in invariant_tokens if not t.startswith(("/*", "//"))]
ext = Parser(invariant_tokens)
name = ext.pop(); ext.pop(":"); expression = ext.expr(); ext.pop(";")
rules[name] = {"kind":"rule", "expr":expression}
member = {"op":"assign","text":"invariants","mode":"+=","args":[{"op":"call","text":"Invariant"}]}
def patch_member(expr, field, op):
    if expr["op"] == op and any(a.get("text") == field for a in walk_expr(expr)):
        if op == "unordered":
            expr["args"].append({"op":"repeat","text":"*","args":[deepcopy(member)]})
        else:
            expr["args"].append(deepcopy(member))
        return True
    return any(patch_member(a,field,op) for a in expr.get("args",[]))
def walk_expr(expr):
    yield expr
    for arg in expr.get("args",[]): yield from walk_expr(arg)
# В BC выбираем самый внутренний unordered, содержащий коллекцию агрегатов.
def patch_bc(expr):
    for arg in expr.get("args",[]):
        if patch_bc(arg): return True
    if expr["op"] == "unordered" and any(a.get("text") == "aggregates" for a in walk_expr(expr)):
        expr["args"].append({"op":"repeat","text":"*","args":[deepcopy(member)]});return True
    return False
assert patch_bc(rules["BoundedContext"]["expr"])
assert patch_member(rules["Aggregate"]["expr"],"domainObjects","alt")
for owner in ["Entity","ValueObject"]:
    assert patch_member(rules[owner]["expr"],"attributes","alt")
out = ROOT/"internal/grammar/grammar.json"
out.write_text(json.dumps(rules, ensure_ascii=False, indent=2)+"\n")
print(f"Compiled {len(rules)} rules to {out}")
