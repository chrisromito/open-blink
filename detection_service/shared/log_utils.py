import logging
import sys


logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)
handler = logging.StreamHandler(sys.stdout)
handler.setLevel(logging.DEBUG)
formatter = logging.Formatter('%(asctime)s %(name)s [%(levelname)s]: %(message)s')
handler.setFormatter(formatter)
logger.addHandler(handler)


def setup_logging(_logger, level=logging.DEBUG):
    _logger.setLevel(level)
    sh = logging.StreamHandler(sys.stdout)
    sh.setLevel(level)
    fmt = logging.Formatter('%(asctime)s %(name)s [%(levelname)s]: %(message)s')
    sh.setFormatter(fmt)
    _logger.addHandler(sh)



