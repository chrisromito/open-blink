from pathlib import Path

from detector.image_detection import get_model, process_images
from shared.perf_utils import block_timer
from main import setup_logging, logger


OUT_PATH = Path('/videos')
ROOT_PATH = Path('/videos/1-1760378857013')

model = get_model()


def test_object_detection(file_names: list[str]):
    timer = block_timer(f'test_object_detection for {len(file_names)} images completed in', log_fn=logger.debug)
    paths: list[str] = [
        str(ROOT_PATH / f_name)
        for f_name in file_names
    ]
    process_images(model, paths)
    timer()


file_name_options = [
    'output-1-1760378856996.jpeg',
    'output-1-1760378857365.jpeg',
    'output-1-1760378857641.jpeg',
    'output-1-1760378858116.jpeg',
    'output-1-1760378858400.jpeg',
    'output-1-1760378858604.jpeg',
    'output-1-1760378859066.jpeg',
    'output-1-1760378859360.jpeg',
    'output-1-1760378859910.jpeg',
    'output-1-1760378860213.jpeg',
    'output-1-1760378860450.jpeg',
    'output-1-1760378860878.jpeg',
    'output-1-1760378861126.jpeg',
    'output-1-1760378861346.jpeg',
    'output-1-1760378861861.jpeg',
    'output-1-1760378861980.jpeg',
    'output-1-1760378862418.jpeg',
    'output-1-1760378862933.jpeg',
    'output-1-1760378863344.jpeg',
    'output-1-1760378864097.jpeg'
]


def test_two():
    test_object_detection(file_name_options[:2])


def test_five():
    test_object_detection(file_name_options[:5])


def test_ten():
    test_object_detection(file_name_options[:10])


def test_all():
    test_object_detection(file_name_options)


def main():
    setup_logging()
    test_two()
    test_five()
    test_ten()
    test_all()


if __name__ == '__main__':
    main()
