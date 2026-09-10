"""Pack app source, not installed dependencies or local session artifacts.

Usage: python3 pack_fixture.py /path/to/todoapp /path/to/todoapp.tar.gz
"""

import argparse
import gzip
import io
from pathlib import Path
import tarfile

ROOT_FILES = (
    ".gitignore", ".npmrc", "README.md", "index.html", "package.json",
    "package-lock.json", "tsconfig.json", "vite.config.ts",
)
SOURCE_SUFFIXES = {".ts", ".tsx", ".css", ".json"}
ASSET_SUFFIXES = {".svg", ".png", ".jpg", ".webp", ".woff", ".woff2"}


def source_files(source: Path) -> list[Path]:
    """Use an allowlist so new tool/cache folders cannot silently bloat runs."""
    if source.is_symlink():
        raise ValueError(f"Refusing symlink: {source}")
    files = []
    for name in ROOT_FILES:
        path = source / name
        if path.is_symlink():
            raise ValueError(f"Refusing symlink: {path}")
        if not path.is_file():
            raise ValueError(f"Missing required file: {path}")
        files.append(path)
    for folder, suffixes in (("src", SOURCE_SUFFIXES), ("public", ASSET_SUFFIXES)):
        root = source / folder
        if root.is_symlink():
            raise ValueError(f"Refusing symlink: {root}")
        if not root.is_dir():
            raise ValueError(f"Missing required directory: {root}")
        for path in sorted(root.rglob("*")):
            if path.is_symlink():
                raise ValueError(f"Refusing symlink: {path}")
            if any(part.startswith(".") for part in path.relative_to(root).parts):
                continue
            if path.is_file() and path.suffix in suffixes:
                files.append(path)
    if source / "src/main.tsx" not in files:
        raise ValueError("Missing required file: src/main.tsx")
    return sorted(files)


def pack(source: Path) -> bytes:
    """Normalize tar and gzip metadata so unchanged source yields identical bytes."""
    files = source_files(source)
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w", format=tarfile.USTAR_FORMAT) as archive:
        for path in files:
            data = path.read_bytes()
            info = tarfile.TarInfo(f"todoapp/{path.relative_to(source).as_posix()}")
            info.size = len(data)
            info.mode = 0o644
            archive.addfile(info, io.BytesIO(data))
    output = io.BytesIO()
    with gzip.GzipFile(fileobj=output, mode="wb", filename="", mtime=0, compresslevel=9) as compressed:
        compressed.write(buffer.getvalue())
    return output.getvalue()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    try:
        data = pack(args.source)
    except (OSError, ValueError) as error:
        parser.exit(1, f"Cannot pack fixture: {error}\n")
    args.output.write_bytes(data)
    print(f"{args.output}: {len(data)} bytes")


if __name__ == "__main__":
    main()
