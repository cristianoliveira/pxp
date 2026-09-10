"""Materialize accepted source, a defect, or a review candidate; no model calls."""

import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import tarfile

ROOT = Path(__file__).resolve().parent


def verify_accepted(root: Path = ROOT / "accepted") -> dict:
    provenance = json.loads((root / "provenance.json").read_text())
    if provenance["decision"] != "accept":
        raise ValueError("Reference has not been accepted")
    for name, expected in provenance["artifact_sha256"].items():
        if Path(name).name != name or name in (".", ".."):
            raise ValueError(f"Unsafe artifact path: {name}")
        path = root / name
        if (
            path.is_symlink()
            or hashlib.sha256(path.read_bytes()).hexdigest() != expected
        ):
            raise ValueError(f"Checksum mismatch: {name}")
    return provenance


def load_source(archive: Path) -> dict[str, bytes]:
    """Read bounded regular files, never follow links or extract archive paths."""
    source = {}
    total = 0
    with tarfile.open(archive) as tar:
        for count, member in enumerate(tar, 1):
            parts = PurePosixPath(member.name).parts
            if count > 128 or not parts or parts[0] != "app" or ".." in parts:
                raise ValueError(f"Unsafe archive member: {member.name}")
            if member.isdir():
                continue
            if not member.isfile() or len(parts) < 2 or member.size > 1_000_000:
                raise ValueError(f"Unsupported archive member: {member.name}")
            total += member.size
            name = PurePosixPath(*parts[1:]).as_posix()
            if total > 4_000_000 or name in source:
                raise ValueError(f"Oversized or duplicate archive member: {name}")
            source[name] = tar.extractfile(member).read()
    if "package-lock.json" not in source or "src/UploadStatus.tsx" not in source:
        raise ValueError("Archive is missing app source or lockfile")
    return source


def variants() -> dict:
    return json.loads((ROOT / "variants.json").read_text())


def alternatives() -> dict:
    return json.loads((ROOT / "alternatives.json").read_text())


def mutate(source: dict[str, bytes], mutation: dict) -> dict[str, bytes]:
    name = mutation["file"]
    if name not in source:
        raise ValueError(f"Missing source file: {name}")
    text = source[name].decode()
    if not mutation["old"] or text.count(mutation["old"]) != 1:
        raise ValueError(f"Mutation must match exactly once in {name}")
    return {**source, name: text.replace(mutation["old"], mutation["new"]).encode()}


def prepare(variant: str, destination: Path) -> None:
    definitions = {**variants(), **alternatives()}
    if variant != "accepted" and variant not in definitions:
        raise ValueError(f"Unknown variant: {variant}")
    verify_accepted()
    source = load_source(ROOT / "accepted/app.tar.gz")
    if variant != "accepted":
        source = mutate(source, definitions[variant])
    destination.mkdir(parents=True, exist_ok=False)
    for name, data in source.items():
        path = destination / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("variant", choices=["accepted", *variants(), *alternatives()])
    parser.add_argument("destination", type=Path, help="New disposable app directory")
    args = parser.parse_args()
    try:
        prepare(args.variant, args.destination)
    except (OSError, ValueError, tarfile.TarError) as error:
        parser.exit(1, f"Cannot prepare control: {error}\n")
    print(args.destination)


if __name__ == "__main__":
    main()
