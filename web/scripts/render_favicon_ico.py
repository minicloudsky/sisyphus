#!/usr/bin/env python3
"""One-shot: render Sisyphus-style DB cylinder icon to public/favicon.ico (Pillow)."""
from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "public" / "favicon.ico"


def draw_icon(size: int) -> Image.Image:
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    s = size / 64.0
    r = max(2, int(round(12 * s)))

    # Diagonal gradient fill inside rounded rect (approximate)
    px = img.load()
    c0 = (79, 70, 229)
    c1 = (124, 58, 237)

    def inside_rounded(x: int, y: int) -> bool:
        if x < 0 or y < 0 or x >= size or y >= size:
            return False
        if x < r and y < r:
            return (x - r + 1) ** 2 + (y - r + 1) ** 2 <= r**2
        if x >= size - r and y < r:
            return (x - (size - r)) ** 2 + (y - r + 1) ** 2 <= r**2
        if x < r and y >= size - r:
            return (x - r + 1) ** 2 + (y - (size - r)) ** 2 <= r**2
        if x >= size - r and y >= size - r:
            return (x - (size - r)) ** 2 + (y - (size - r)) ** 2 <= r**2
        return True

    for y in range(size):
        for x in range(size):
            if not inside_rounded(x, y):
                continue
            t = (x + y) / max(1, 2 * (size - 1))
            rr = int(c0[0] + (c1[0] - c0[0]) * t)
            gg = int(c0[1] + (c1[1] - c0[1]) * t)
            bb = int(c0[2] + (c1[2] - c0[2]) * t)
            px[x, y] = (rr, gg, bb, 255)

    draw = ImageDraw.Draw(img)
    cx, white = 32 * s, (255, 255, 255)

    def ell(ex: float, ey: float, erx: float, ery: float, alpha: float) -> None:
        a = int(round(255 * alpha))
        f = white + (a,)
        draw.ellipse([cx - erx, ey - ery, cx + erx, ey + ery], fill=f)

    def rect(x0: float, y0: float, x1: float, y1: float, alpha: float) -> None:
        a = int(round(255 * alpha))
        f = white + (a,)
        draw.rectangle([x0, y0, x1, y1], fill=f)

    ell(32 * s, 20 * s, 14 * s, 5 * s, 0.95)
    rect(18 * s, 20 * s, 46 * s, 32 * s, 0.85)
    ell(32 * s, 32 * s, 14 * s, 5 * s, 0.95)
    rect(18 * s, 32 * s, 46 * s, 44 * s, 0.75)
    ell(32 * s, 44 * s, 14 * s, 5 * s, 0.95)
    return img


def main() -> None:
    sizes = (16, 32, 48, 64)
    frames = [draw_icon(sz) for sz in sizes]
    OUT.parent.mkdir(parents=True, exist_ok=True)
    frames[0].save(
        OUT,
        format="ICO",
        append_images=frames[1:],
    )
    print(f"Wrote {OUT}")


if __name__ == "__main__":
    main()
