#!/usr/bin/env python3
import json
from pathlib import Path

import cv2
import numpy as np


ROOT = Path(__file__).resolve().parents[1]
PICTURE_DIR = ROOT / "picture"
OUTPUT_DIR = PICTURE_DIR / "synthetic"
MANIFEST_PATH = OUTPUT_DIR / "manifest.json"

TEMPLATES = [
    {
        "name": "incubator_low",
        "base": PICTURE_DIR / "孵蛋页面_单蛋加速91_低分辨率.png",
        "bg": (45, 45, 45),
        "size_rect": (220, 132, 324, 160),
        "weight_rect": (220, 170, 312, 196),
        "size_pos": (228, 139),
        "weight_pos": (228, 176),
        "sources": [
            {"image": PICTURE_DIR / "孵蛋页面_单蛋加速91_低分辨率.png", "rect": (220, 132, 285, 160), "text": "0.23"},
            {"image": PICTURE_DIR / "孵蛋页面_单蛋加速91_低分辨率.png", "rect": (220, 170, 279, 196), "text": "2.75"},
            {"image": PICTURE_DIR / "孵蛋页面_单蛋加速91_低分辨率.png", "rect": (224, 96, 276, 121), "text": "91%"},
            {"image": PICTURE_DIR / "孵蛋页面3.png", "rect": (216, 151, 266, 177), "text": "0.17"},
            {"image": PICTURE_DIR / "孵蛋页面3.png", "rect": (216, 184, 283, 210), "text": "2.775"},
        ],
        "cases": [
            {"id": "common", "size": "0.17", "weight": "2.775"},
            {"id": "boost", "size": "0.23", "weight": "2.750"},
            {"id": "var1", "size": "0.19", "weight": "2.715"},
            {"id": "var2", "size": "0.21", "weight": "2.795"},
        ],
    },
    {
        "name": "detail_dark",
        "base": PICTURE_DIR / "蛋详细页面_神奇的蛋_低重量.png",
        "bg": (45, 45, 45),
        "size_rect": (72, 240, 190, 282),
        "weight_rect": (240, 240, 360, 282),
        "size_pos": (82, 247),
        "weight_pos": (252, 247),
        "sources": [
            {"image": PICTURE_DIR / "蛋详细页面_神奇的蛋_低重量.png", "rect": (78, 248, 166, 282), "text": "0.21"},
            {"image": PICTURE_DIR / "蛋详细页面_神奇的蛋_低重量.png", "rect": (248, 248, 349, 282), "text": "0.309"},
            {"image": PICTURE_DIR / "蛋详细页面_神奇的蛋_低重量.png", "rect": (404, 248, 550, 281), "text": "2026-04-12"},
            {"image": PICTURE_DIR / "蛋详细页面_粉星仔蛋.png", "rect": (244, 259, 345, 291), "text": "5.04"},
            {"image": PICTURE_DIR / "蛋详细页面_豆丁鱼蛋.png", "rect": (246, 243, 360, 276), "text": "0.558"},
            {"image": PICTURE_DIR / "蛋详细页面_豆丁鱼蛋.png", "rect": (78, 243, 166, 276), "text": "0.17"},
        ],
        "cases": [
            {"id": "low_weight", "size": "0.21", "weight": "0.309"},
            {"id": "fish", "size": "0.17", "weight": "0.558"},
            {"id": "common", "size": "0.17", "weight": "2.775"},
            {"id": "boost", "size": "0.23", "weight": "2.750"},
            {"id": "detail", "size": "0.22", "weight": "6.181"},
            {"id": "heavy", "size": "0.33", "weight": "11.573"},
        ],
    },
]


def binarize(crop):
    gray = cv2.cvtColor(crop, cv2.COLOR_BGR2GRAY)
    mask = cv2.inRange(gray, 150, 255)
    return mask


def extract_components(crop):
    mask = binarize(crop)
    num, _, stats, _ = cv2.connectedComponentsWithStats(mask, 8)
    items = []
    for idx in range(1, num):
        x, y, w, h, area = stats[idx]
        if area < 4 or w < 2 or h < 2:
            continue
        items.append((x, y, w, h, area))
    items.sort(key=lambda item: item[0])
    return mask, items


def extract_glyphs(source):
    image = cv2.imread(str(source["image"]))
    if image is None:
        raise RuntimeError(f"failed to load {source['image']}")
    x1, y1, x2, y2 = source["rect"]
    crop = image[y1:y2, x1:x2]
    mask, components = extract_components(crop)
    text = source["text"]
    if len(components) < len(text):
        raise RuntimeError(f"not enough glyph components for {source['image']} {text}: {len(components)}")
    components = components[: len(text)]
    glyphs = []
    for char, (x, y, w, h, _area) in zip(text, components):
        if char in {"-", "%"}:
            continue
        rgba = np.zeros((h, w, 4), dtype=np.uint8)
        rgba[:, :, :3] = crop[y : y + h, x : x + w]
        rgba[:, :, 3] = mask[y : y + h, x : x + w]
        glyphs.append((char, rgba))
    return glyphs


def build_atlas(template):
    atlas = {}
    for source in template["sources"]:
        for char, glyph in extract_glyphs(source):
            atlas.setdefault(char, glyph)
    return atlas


def compose_value(text, atlas):
    glyphs = []
    for char in text:
        glyph = atlas.get(char)
        if glyph is None:
            raise RuntimeError(f"missing glyph for {char}")
        glyphs.append(glyph)

    max_h = max(g.shape[0] for g in glyphs)
    spacing = 3
    width = sum(g.shape[1] for g in glyphs) + spacing * (len(glyphs) - 1)
    canvas = np.zeros((max_h, width, 4), dtype=np.uint8)
    x = 0
    for glyph in glyphs:
        y = max_h - glyph.shape[0]
        alpha = glyph[:, :, 3:4] / 255.0
        canvas[y : y + glyph.shape[0], x : x + glyph.shape[1], :3] = (
            glyph[:, :, :3] * alpha
            + canvas[y : y + glyph.shape[0], x : x + glyph.shape[1], :3] * (1 - alpha)
        ).astype(np.uint8)
        canvas[y : y + glyph.shape[0], x : x + glyph.shape[1], 3] = np.maximum(
            canvas[y : y + glyph.shape[0], x : x + glyph.shape[1], 3], glyph[:, :, 3]
        )
        x += glyph.shape[1] + spacing
    return canvas


def paste_value(img, value_rgba, rect, pos, bg):
    x1, y1, x2, y2 = rect
    cv2.rectangle(img, (x1, y1), (x2, y2), bg, -1)

    h, w = value_rgba.shape[:2]
    px, py = pos
    y = py + max((y2 - y1 - h) // 2, 0)
    x = px
    alpha = value_rgba[:, :, 3:4] / 255.0
    target = img[y : y + h, x : x + w]
    if target.shape[0] != h or target.shape[1] != w:
        raise RuntimeError("target region too small for pasted value")
    img[y : y + h, x : x + w] = (value_rgba[:, :, :3] * alpha + target * (1 - alpha)).astype(np.uint8)


def main():
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    manifest = []
    for template in TEMPLATES:
        atlas = build_atlas(template)
        base = cv2.imread(str(template["base"]))
        if base is None:
            raise RuntimeError(f"failed to read template: {template['base']}")
        for case in template["cases"]:
            img = base.copy()
            size_rgba = compose_value(case["size"], atlas)
            weight_rgba = compose_value(case["weight"], atlas)
            paste_value(img, size_rgba, template["size_rect"], template["size_pos"], template["bg"])
            paste_value(img, weight_rgba, template["weight_rect"], template["weight_pos"], template["bg"])

            name = f"{template['name']}_{case['id']}.png"
            out_path = OUTPUT_DIR / name
            cv2.imwrite(str(out_path), img)
            manifest.append(
                {
                    "image": str(out_path),
                    "template": template["name"],
                    "case": case["id"],
                    "size": float(case["size"]),
                    "weight": float(case["weight"]),
                }
            )

    MANIFEST_PATH.write_text(json.dumps(manifest, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"generated {len(manifest)} cases")
    print(MANIFEST_PATH)


if __name__ == "__main__":
    main()
