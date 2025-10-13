from pathlib import Path

from detector.image_detection import get_model, process_image


test_videos_path = Path("./test_videos")

test_images = [
    (test_videos_path / "BadgerFoxImage.jpg", 'fox'),
    (test_videos_path / "charchar.jpg", 'dog'),
    (test_videos_path / "Racoon.jpg", 'racoon'),
]


def test_process_image(model, image_path: str, expected_label: str = None):
    image, detections, categories = process_image(model, image_path)
    out_path = image_path.replace('.jpg', '_output.jpg')
    image.save(out_path)
    print(f"Detected {len(detections)} objects in {image_path}")
    print(f"Categories: {categories}")
    for detection in detections:
        print(f" - {detection['label']} ({detection['confidence']})")
    labels = [d.get('label') for d in detections]
    # assert expected_label in labels, f"Expected {expected_label} but got {labels}"


def run():
    model = get_model()
    for image_path, expected_label in test_images:
        test_process_image(model, str(image_path), expected_label)


if __name__ == '__main__':
    run()
