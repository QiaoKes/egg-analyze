import json
import sys
import traceback

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
    engine = RapidOCR(
        det_limit_side_len=960,
        det_box_thresh=0.42,
        max_side_len=2560,
        width_height_ratio=12,
    )
    for raw in sys.stdin:
        raw = raw.strip()
        if not raw:
            continue
        try:
            payload = json.loads(raw)
            path = payload["path"]
            result, _ = engine(path)
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
            print(json.dumps({"ok": True, "lines": lines}, ensure_ascii=False), flush=True)
        except Exception as exc:
            print(
                json.dumps(
                    {
                        "ok": False,
                        "error": str(exc),
                        "traceback": traceback.format_exc(limit=1),
                    },
                    ensure_ascii=False,
                ),
                flush=True,
            )


if __name__ == "__main__":
    main()
