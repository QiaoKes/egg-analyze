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
    if hasattr(sys.stdin, "reconfigure"):
        sys.stdin.reconfigure(encoding="utf-8")
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")

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
            mode = payload.get("mode", "full")
            if mode == "rec_only":
                result, _ = engine(path, use_det=False, use_cls=False, use_rec=True)
            else:
                result, _ = engine(path)
            lines = []
            if mode == "rec_only":
                for item in result or []:
                    text = item[0]
                    lines.append(
                        {
                            "Text": text,
                            "X": 0,
                            "Y": 0,
                            "Width": 0,
                            "Height": 0,
                        }
                    )
            else:
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
            print(json.dumps({"ok": True, "lines": lines}), flush=True)
        except Exception as exc:
            print(
                json.dumps(
                    {
                        "ok": False,
                        "error": str(exc),
                        "traceback": traceback.format_exc(limit=1),
                    },
                ),
                flush=True,
            )


if __name__ == "__main__":
    main()
