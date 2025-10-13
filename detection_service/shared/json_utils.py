from pathlib import Path
import json
from typing import Union


def read_json(file_path: Union[Path, str]) -> Union[dict, list]:
    with open(file_path) as f:
        return json.load(f)


def write_json(file_path: Union[Path, str], data: Union[dict, list]) -> None:
    with open(file_path, 'w+') as f:
        json.dump(data, f, indent=4)


def safe_stringify(obj: dict | list) -> str:
    return json.dumps(obj, indent=4, sort_keys=True, default=str)


def safe_pojo(obj: dict | list) -> dict | list:
    """
    Any -> str -> dict | list
    """
    return json.loads(safe_stringify(obj))