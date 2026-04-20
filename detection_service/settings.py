from pathlib import Path


app_dir_path = Path(__file__).resolve().parent
ROOT_PATH = app_dir_path.parent
MODEL_PATH = ROOT_PATH / 'models'
YOLO_PT = MODEL_PATH / 'yolo26n.pt'


IMAGE_SOCKET_PORT = 8000
JSON_SOCKET_PORT = 5000