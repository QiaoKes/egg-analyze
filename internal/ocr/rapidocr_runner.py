import json
import sys

from rapidocr_onnxruntime import RapidOCR


def bounds(box):
    xs = [int(point[0]) for point in box]
    ys = [int(point[1]) for point in box]
    left = min(xs)
    top = min(ys)
    right = max(xs)
    bottom = max(ys)
    return left, top, right - left, bottom - top


def main():
    engine = RapidOCR()
    result, _ = engine(sys.argv[1])
    lines = []
    for item in result or []:
        box, text, _score = item
        x, y, width, height = bounds(box)
        lines.append(
            {
                "Text": text,
                "X": x,
                "Y": y,
                "Width": width,
                "Height": height,
            }
        )
    print(json.dumps(lines, ensure_ascii=False))


if __name__ == "__main__":
    main()
