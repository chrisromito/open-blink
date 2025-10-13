from datetime import datetime


def get_epoch() -> int:
    return int(datetime.now().timestamp() * 1000)
