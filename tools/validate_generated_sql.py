#!/usr/bin/env python3
"""Check the generated DDL before it ever reaches a server.

Runs three passes over database/s4hana:

1. T-SQL parse of every batch (sqlglot, tsql dialect) - catches syntax errors.
2. Referential checks - every foreign key column exists, and every referenced
   column list is exactly the primary key or the unique business key of the
   target table.
3. Identifier checks - unique constraint names, names within 128 characters.

    python3 tools/validate_generated_sql.py
"""

from __future__ import annotations

import collections
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))

from catalogue_model import ROOT, load_tables, resolve_foreign_keys  # noqa: E402

SQL_DIR = ROOT / "database" / "s4hana"
CONSTRAINT = re.compile(r"CONSTRAINT \[([A-Za-z0-9_]+)\]")
GO_SPLIT = re.compile(r"^GO\s*$", re.MULTILINE)


def parse_batches() -> list[str]:
    try:
        import sqlglot
    except ImportError:
        print("  sqlglot not installed - skipping the parse pass")
        return []
    problems: list[str] = []
    for path in sorted(SQL_DIR.glob("*.sql")):
        if path.name == "run_all.sql":
            continue  # sqlcmd :r directives, not T-SQL
        text = path.read_text(encoding="utf-8-sig")
        for number, batch in enumerate(GO_SPLIT.split(text), start=1):
            if not batch.strip():
                continue
            try:
                sqlglot.parse(batch, dialect="tsql")
            except Exception as error:  # noqa: BLE001 - report, do not raise
                first = batch.strip().splitlines()[0][:90]
                problems.append(f"{path.name} batch {number}: {error} | {first}")
    return problems


def check_references() -> list[str]:
    tables = load_tables()
    lookup = {t.full_name: t for t in tables}
    foreign_keys, _ = resolve_foreign_keys(tables)
    problems: list[str] = []
    for fk in foreign_keys:
        for column in fk.columns:
            if fk.table.field(column) is None:
                problems.append(f"{fk.name}: {fk.table.full_name} has no [{column}]")
        target = lookup[f"{fk.target_schema}.{fk.target_table}"]
        pk = [f.name for f in target.primary_key]
        ak = [f.name for f in target.alternate_key]
        if fk.target_columns not in (pk, ak):
            problems.append(
                f"{fk.name}: {target.full_name}{fk.target_columns} is neither the "
                f"primary key {pk} nor the business key {ak}"
            )
        types_local = [fk.table.field(c).sql_type for c in fk.columns]  # type: ignore[union-attr]
        types_target = [target.field(c).sql_type for c in fk.target_columns]  # type: ignore[union-attr]
        if types_local != types_target:
            problems.append(
                f"{fk.name}: type mismatch {types_local} vs {types_target}"
            )
    return problems


INSERT = re.compile(
    r"INSERT\s+INTO\s+(?:\[?(\w+)\]?\.\[?(\w+)\]?)\s*\(([^)]*(?:\([^)]*\)[^)]*)*)\)\s*"
    r"(?:VALUES|SELECT|OUTPUT)",
    re.IGNORECASE | re.DOTALL,
)


def check_inserts() -> list[str]:
    """Every seeded column must exist, and every column the database cannot
    fill by itself must be supplied."""
    tables = {t.full_name: t for t in load_tables()}
    problems: list[str] = []
    for path in sorted(SQL_DIR.glob("9*_seed*.sql")):
        text = path.read_text(encoding="utf-8-sig")
        for schema, name, column_list, in [
            (m.group(1), m.group(2), m.group(3)) for m in INSERT.finditer(text)
        ]:
            full_name = f"{schema}.{name}"
            table = tables.get(full_name)
            if table is None:
                if not full_name.startswith("#"):
                    problems.append(f"{path.name}: unknown table {full_name}")
                continue
            columns = [c.strip().strip("[]") for c in column_list.split(",") if c.strip()]
            for column in columns:
                if table.field(column) is None:
                    problems.append(f"{path.name}: {full_name} has no column {column}")
            supplied = set(columns)
            for f in table.fields:
                if f.nullable or f.name in supplied:
                    continue
                # The database fills these on its own.
                if f.sql_type == "rowversion" or (f.is_pk and f.name == "Id"):
                    continue
                if f.name == "CreatedAt" or f.sql_type == "bit":
                    continue
                problems.append(
                    f"{path.name}: {full_name}.{f.name} is required but not supplied"
                )
    return problems


def split_top_level(text: str) -> list[str]:
    """Split on commas that are not inside parentheses or a string literal."""
    parts, current, depth, in_string = [], [], 0, False
    index = 0
    while index < len(text):
        char = text[index]
        if in_string:
            if char == "'":
                if index + 1 < len(text) and text[index + 1] == "'":
                    current.append("''")
                    index += 2
                    continue
                in_string = False
            current.append(char)
        elif char == "'":
            in_string = True
            current.append(char)
        elif char == "(":
            depth += 1
            current.append(char)
        elif char == ")":
            depth -= 1
            current.append(char)
        elif char == "," and depth == 0:
            parts.append("".join(current).strip())
            current = []
        else:
            current.append(char)
        index += 1
    if "".join(current).strip():
        parts.append("".join(current).strip())
    return parts


STRING_LITERAL = re.compile(r"^N?'((?:[^']|'')*)'$", re.DOTALL)
BLOCK_COMMENT = re.compile(r"/\*.*?\*/", re.DOTALL)
LINE_COMMENT = re.compile(r"--[^\n]*")


def strip_comments(text: str) -> str:
    return LINE_COMMENT.sub("", BLOCK_COMMENT.sub("", text))


def value_tuples(text: str) -> list[str]:
    """Top-level parenthesised groups - one per VALUES row. Depth aware, so a
    nested call such as DATEFROMPARTS(2026, 1, 1) does not end a row."""
    tuples, depth, start, in_string = [], 0, None, False
    for index, char in enumerate(text):
        if in_string:
            if char == "'":
                in_string = False
            continue
        if char == "'":
            in_string = True
        elif char == "(":
            if depth == 0:
                start = index + 1
            depth += 1
        elif char == ")":
            depth -= 1
            if depth == 0 and start is not None:
                tuples.append(text[start:index])
                start = None
        elif char == ";" and depth == 0:
            break
    return tuples


def select_list(text: str) -> str | None:
    """
    The projection of an INSERT ... SELECT, up to its FROM or terminator.

    <paramref name="text"/> begins just after the SELECT keyword, which the
    INSERT pattern has already consumed.
    """
    start = 0
    depth = 0
    index = start
    while index < len(text):
        character = text[index]
        if character == "'":
            index += 1
            while index < len(text):
                if text[index] == "'":
                    if text[index + 1 : index + 2] == "'":
                        index += 2
                        continue
                    break
                index += 1
        elif character == "(":
            depth += 1
        elif character == ")":
            depth -= 1
        elif depth == 0:
            if character == ";":
                return text[start:index]
            if re.match(r"\bFROM\b", text[index:], re.IGNORECASE):
                return text[start:index]
        index += 1

    return None


def select_literal_problems(file_name, table, match, text) -> list[str]:
    """
    Checks the constant literals of an INSERT ... SELECT against their columns.

    SQL Server catches an over-long literal here too - by raising error 2628
    half way through the install, with the script already partly applied. The
    point of this pass is to catch it before that.
    """
    projection = select_list(text[match.end() :])
    if projection is None:
        return []

    columns = [c.strip().strip("[]") for c in match.group(3).split(",") if c.strip()]
    expressions = split_top_level(projection)
    if len(expressions) != len(columns):
        return [
            f"{file_name}: {table.full_name} selects {len(expressions)} expressions "
            f"for {len(columns)} columns"
        ]

    problems: list[str] = []
    for column_name, expression in zip(columns, expressions):
        field = table.field(column_name)
        literal = STRING_LITERAL.match(expression.strip())
        if field is None or literal is None:
            continue
        limit = field.max_length
        actual = len(literal.group(1).replace("''", "'"))
        if limit is not None and actual > limit:
            problems.append(
                f"{file_name}: {table.full_name}.{column_name} is {field.sql_type} "
                f"but the selected literal is {actual} characters"
            )
    return problems


def check_insert_arity() -> list[str]:
    """Each VALUES row must supply exactly one value per column, and a string
    literal must fit the column it goes into."""
    tables = {t.full_name: t for t in load_tables()}
    problems: list[str] = []
    for path in sorted(SQL_DIR.glob("9*_seed*.sql")):
        text = path.read_text(encoding="utf-8-sig")
        for match in INSERT.finditer(text):
            table = tables.get(f"{match.group(1)}.{match.group(2)}")
            if table is None:
                continue
            if not text[match.start() : match.end()].rstrip().upper().endswith("VALUES"):
                problems.extend(
                    select_literal_problems(path.name, table, match, text)
                )
                continue
            columns = [c.strip().strip("[]") for c in match.group(3).split(",") if c.strip()]
            for row_number, row in enumerate(
                value_tuples(strip_comments(text[match.end() :])), start=1
            ):
                values = split_top_level(row)
                where = f"{path.name}: {table.full_name} row {row_number}"
                if len(values) != len(columns):
                    problems.append(
                        f"{where}: {len(values)} values for {len(columns)} columns"
                    )
                    continue
                for column_name, value in zip(columns, values):
                    field = table.field(column_name)
                    literal = STRING_LITERAL.match(value)
                    if field is None or literal is None:
                        continue
                    limit = field.max_length
                    actual = len(literal.group(1).replace("''", "'"))
                    if limit is not None and actual > limit:
                        problems.append(
                            f"{where}: {column_name} is {field.sql_type} but the "
                            f"value is {actual} characters"
                        )
    return problems


def check_identifiers() -> list[str]:
    problems: list[str] = []
    seen: collections.Counter[str] = collections.Counter()
    for path in sorted(SQL_DIR.glob("*.sql")):
        for name in CONSTRAINT.findall(path.read_text(encoding="utf-8-sig")):
            seen[name] += 1
            if len(name) > 128:
                problems.append(f"identifier longer than 128 characters: {name}")
    # DEFAULT constraints appear once per table, PK/UQ/FK names must be unique.
    for name, count in seen.items():
        if count > 1:
            problems.append(f"constraint name used {count} times: {name}")
    return problems


# A schema-qualified table name written without brackets. sqlglot parses
# sec.User happily; SQL Server does not, because User is a reserved word. The
# cheapest defence is to require brackets everywhere rather than to keep a list
# of which of the 228 table names happen to be keywords this release.
BARE_REFERENCE = re.compile(
    r"(?<![\[\w.'])(org|cfg|mdm|fin|co|wf|sec|audit|rpt|intg)\.([A-Za-z_][A-Za-z0-9_]*)"
)


def strip_strings_and_comments(text: str) -> str:
    """Blanks out literals and comments so only executable code is scanned."""
    out: list[str] = []
    index = 0
    length = len(text)
    while index < length:
        if text[index] == "'":
            end = index + 1
            while end < length:
                if text[end] == "'":
                    if text[end + 1 : end + 2] == "'":
                        end += 2
                        continue
                    end += 1
                    break
                end += 1
            out.append(" " * (end - index))
            index = end
        elif text[index : index + 2] == "--":
            end = text.find("\n", index)
            end = length if end < 0 else end
            out.append(" " * (end - index))
            index = end
        elif text[index : index + 2] == "/*":
            end = text.find("*/", index + 2)
            end = length if end < 0 else end + 2
            out.append(" " * (end - index))
            index = end
        else:
            out.append(text[index])
            index += 1
    return "".join(out)


def check_bracketing() -> list[str]:
    tables = {(t.schema, t.name) for t in load_tables()}
    problems: list[str] = []

    for path in sorted(SQL_DIR.glob("*.sql")):
        code = strip_strings_and_comments(path.read_text(encoding="utf-8-sig"))
        bare = {
            f"{m.group(1)}.{m.group(2)}"
            for m in BARE_REFERENCE.finditer(code)
            if (m.group(1), m.group(2)) in tables
        }
        problems.extend(
            f"{path.name}: {name} is not bracketed - write [schema].[Table]"
            for name in sorted(bare)
        )

    return problems


def main() -> None:
    failures = 0
    for title, problems in (
        ("T-SQL parse", parse_batches()),
        ("references", check_references()),
        ("identifiers", check_identifiers()),
        ("bracketing", check_bracketing()),
        ("seed inserts", check_inserts()),
        ("seed values", check_insert_arity()),
    ):
        if problems:
            failures += len(problems)
            print(f"{title}: {len(problems)} problem(s)")
            for problem in problems[:20]:
                print(f"    {problem}")
            if len(problems) > 20:
                print(f"    ... and {len(problems) - 20} more")
        else:
            print(f"{title}: ok")
    sys.exit(1 if failures else 0)


if __name__ == "__main__":
    main()
