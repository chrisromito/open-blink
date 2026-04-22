import json
import logging
import socket
import struct
from io import BytesIO

import requests
from PIL import Image

from shared.log_utils import setup_logging
from shared.perf_utils import block_timer


def load_test_image() -> Image:
    print(f'Downloading bus image...')
    resp = requests.get("https://ultralytics.com/images/bus.jpg")
    if resp.status_code != 200:
        raise Exception('Could not fetch image data')
    data = BytesIO(resp.content)
    return Image.open(data)


def send_test_image(host: str, port: int, img: Image):
    """
    Takes 'img' & sends it to 'endpoint' using the struct defined in http_server.py
    Args:
        host: Host address (e.g., '192.168.0.65')
        port: Port number (e.g., 8000)
        img: PIL Image to send
    Returns:
    """
    endpoint = f'{host}:{port}'
    api_timer = block_timer(endpoint)
    # Convert PIL Image to JPEG bytes
    img_buffer = BytesIO()
    img.save(img_buffer, format='JPEG')
    image_bytes = img_buffer.getvalue()
    # Create socket connection and send data using the same protocol as http_server.py
    sock = socket.create_connection((host, port))
    try:
        # Send length-prefixed JPEG data
        sock.sendall(struct.pack('>I', len(image_bytes)))
        sock.sendall(image_bytes)

        # Receive length-prefixed response
        raw_len = sock.recv(4)
        msg_len = struct.unpack('>I', raw_len)[0]
        response_data = sock.recv(msg_len).decode('utf-8')
        # Parse JSON response
        return json.loads(response_data)
    finally:
        sock.close()
        api_timer()


def test_inference():
    logger = logging.getLogger(__name__)
    setup_logging(logger)
    image = load_test_image()
    total_timer = block_timer('test_inference')
    for _ in range(100):
        send_test_image('192.168.0.65', 8000, image)
    total_timer()

if __name__ == '__main__':
    test_inference()