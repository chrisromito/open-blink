import asyncio
import logging

import settings
from detector.process_image import Processor
from server.http_server import run
from shared.log_utils import setup_logging


logger = logging.getLogger(__name__)


async def main_asyncio_server():
    setup_logging(logger, level=logging.DEBUG)
    processor = Processor.from_path(settings.YOLO_PT)
    logger.info(f'Server starting at 0.0.0.0:{settings.IMAGE_SOCKET_PORT}')
    return await run(processor, '0.0.0.0', port=settings.IMAGE_SOCKET_PORT)


if __name__ == "__main__":
    asyncio.run(main_asyncio_server())
