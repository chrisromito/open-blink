import logging
import time
from shared.log_utils import logger


def block_timer(message: str, log_fn=logger.debug):
    """
    timer_one = block_timer('whole function')
    timer_two = block_timer('first step')
    my_expensive_function()
    timer_two()
    for n in range(100):
        my_expensive_function()
    timer_one()
    """
    start = time.time()

    def end_timer():
        end = time.time()
        diff = end - start
        log_fn(f'{message}: {diff:.6f}s')

    return end_timer
