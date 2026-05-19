import json
from pathlib import Path


def read_json(file_path: Path | str) -> dict | list:
    with open(file_path) as f:
        return json.load(f)


def write_json(file_path: Path | str, data: dict | list) -> None:
    with open(file_path, "w+") as f:
        json.dump(data, f, indent=4)


def safe_stringify(obj: dict | list) -> str:
    return json.dumps(obj, indent=4, sort_keys=True, default=str)


def safe_pojo(obj: dict | list) -> dict | list:
    """
    Any -> str -> dict | list
    """
    return json.loads(safe_stringify(obj))
