"""Apparent bytes in one owned tree; disappearing entries are observable transients."""
import os
import stat


def apparent_bytes(root, missing):
    raw_root = os.fsencode(root)
    seen = set()

    def vanished(operation, path, error):
        missing(operation, raw_root, path, error)

    def directory(fd, path):
        info = os.fstat(fd)
        identity = (info.st_dev, info.st_ino)
        if identity in seen:
            return 0
        seen.add(identity)
        # Installed uutils du -sb excludes directory headers from apparent bytes.
        total = 0
        try:
            entries = os.scandir(fd)
        except FileNotFoundError as error:
            vanished('scandir', path, error)
            return total
        with entries:
            while True:
                try:
                    entry = next(entries)
                except StopIteration:
                    break
                except FileNotFoundError as error:
                    vanished('scandir-next', path, error)
                    break
                raw_path = os.path.join(path, os.fsencode(entry.name))
                try:
                    info = entry.stat(follow_symlinks=False)
                except FileNotFoundError as error:
                    vanished('lstat', raw_path, error)
                    continue
                if stat.S_ISDIR(info.st_mode):
                    try:
                        child = os.open(entry.name, os.O_RDONLY | os.O_DIRECTORY |
                                        os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=fd)
                    except FileNotFoundError as error:
                        vanished('open-directory', raw_path, error)
                        continue
                    try:
                        total += directory(child, raw_path)
                    finally:
                        os.close(child)
                else:
                    identity = (info.st_dev, info.st_ino)
                    if identity not in seen:
                        seen.add(identity)
                        total += info.st_size
        return total

    try:
        info = os.stat(raw_root, follow_symlinks=False)
    except FileNotFoundError as error:
        vanished('lstat-root', raw_root, error)
        return 0
    if not stat.S_ISDIR(info.st_mode):
        return info.st_size
    try:
        fd = os.open(raw_root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)
    except FileNotFoundError as error:
        vanished('open-root', raw_root, error)
        return 0
    try:
        return directory(fd, raw_root)
    finally:
        os.close(fd)
