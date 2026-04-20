import asyncio
import json
import logging
import socket
import struct

import cv2
import numpy as np

import settings
from detector.detection_types import Detection
from detector.process_image import Processor
from shared.log_utils import setup_logging


logger = logging.getLogger(__name__)
setup_logging(logger)


async def run(processor: Processor, host: str = '0.0.0.0', port: int = settings.IMAGE_SOCKET_PORT):
    logger.info(f'Server starting at {host}:{settings.IMAGE_SOCKET_PORT}')
    handler = image_handler(processor)
    server = await asyncio.start_server(handler, host, port)
    async with server:
        await server.serve_forever()


def image_handler(processor: Processor):
    async def _image_handler(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
        try:
            while True:
                # Read message length (4 bytes)
                raw_len = await reader.readexactly(4)
                msg_len = struct.unpack(">I", raw_len)[0]
                # Read JPEG bytes
                data: bytes = await reader.readexactly(msg_len)
                logger.debug(f'http_server -> image_handler() -> received image')

                # Decode image
                image = cv2.imdecode(
                    np.frombuffer(data, dtype=np.uint8),
                    cv2.IMREAD_COLOR
                )
                logger.debug(f'http_server -> image_handler() -> running inference on image')

                results: list[Detection] = processor.predict(image)  # run_detection(image)
                # Encode JSON response
                response = json.dumps(results).encode("utf-8")
                logger.debug(f'Results: {results}')
                # Send length-prefixed response
                writer.write(struct.pack(">I", len(response)))
                writer.write(response)
                await writer.drain()
        except asyncio.IncompleteReadError:
            pass
        except Exception as err:
            logger.exception(err)
            raise err
        finally:
            writer.close()
            await writer.wait_closed()

    return _image_handler


def send_test_image(image_bytes, host: str = '0.0.0.0', port: int = settings.IMAGE_SOCKET_PORT):
    sock = socket.create_connection((host, port))
    logger.debug(f'Sending image to: {host}:{port}')
    sock.sendall(struct.pack('>I', len(image_bytes)))
    sock.sendall(image_bytes)
    logger.debug('Receiving response')
    raw_len = sock.recv(4)
    msg_len = struct.unpack('>I', raw_len)[0]
    data = sock.recv(msg_len).decode()
    logger.debug(data)


def test_image_server():
    with open('/app/test/test-image.jpeg', 'rb') as f:
        image_bytes = f.read()
    send_test_image(image_bytes)
