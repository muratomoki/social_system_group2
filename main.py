import base64
import json
import requests
from io import BytesIO
from PIL import Image, ImageOps, ImageFilter

IMAGE_PATH = "receipt.webp"
MODEL = "gemma4:e4b"
OLLAMA_URL = "http://localhost:11434/api/generate"

# 返してほしい形は「商品だけ」
SCHEMA = {
    "type": "object",
    "properties": {
        "items": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "name": {"type": ["string", "null"]},
                    "qty": {"type": ["string", "null"]},
                    "unit_price": {"type": ["string", "null"]},
                    "amount": {"type": ["string", "null"]}
                },
                "required": ["name", "qty", "unit_price", "amount"]
            }
        }
    },
    "required": ["items"]
}

PROMPT = """
この画像はレシートの一部または全体です。
店ごとに形は違いますが、購入した商品・サービスの明細行だけを抽出してください。

必ず指定のJSONだけを返してください。
説明文は不要です。

items には「商品名がある購入行」だけを入れてください。
次の情報は items に入れないでください。
店名、住所、電話番号、日時、レジ番号、担当者、会員情報、
小計、税、合計、預り、釣り、支払方法、カード、現金、ポイント、注意書き。

qty は画像に数量が明記されている場合だけ入れてください。
unit_price は画像に単価が明記されている場合だけ入れてください。
amount はその商品行の金額を入れてください。
金額が1つしかない場合は amount に入れてください。

値引き専用行、クーポン行、ポイント行は商品ではないので items に入れないでください。
読めない値は null にしてください。
推測で補完しないでください。
同じ商品が別の画像にも見えても、そのまま返してください。
"""

def preprocess_image(path: str) -> Image.Image:
    img = Image.open(path).convert("L")
    img = ImageOps.autocontrast(img)
    w, h = img.size
    img = img.resize((w * 2, h * 2))
    img = img.filter(ImageFilter.SHARPEN)
    return img

def split_vertical(img: Image.Image, band_height: int = 1400, overlap: int = 220):
    w, h = img.size
    parts = []
    top = 0
    idx = 0

    while top < h:
        bottom = min(h, top + band_height)
        part = img.crop((0, top, w, bottom))
        parts.append((idx, top, bottom, part))

        if bottom == h:
            break

        top = bottom - overlap
        idx += 1

    return parts

def image_to_b64(img: Image.Image) -> str:
    buf = BytesIO()
    img.save(buf, format="PNG")
    return base64.b64encode(buf.getvalue()).decode("utf-8")

def call_model(image_b64: str) -> dict:
    payload = {
        "model": MODEL,
        "prompt": PROMPT,
        "images": [image_b64],
        "format": SCHEMA,
        "stream": False,
        "think": False
    }

    r = requests.post(OLLAMA_URL, json=payload, timeout=600)
    r.raise_for_status()

    text = r.json()["response"]
    return json.loads(text)

def norm(v):
    if v is None:
        return None
    v = str(v).strip()
    v = " ".join(v.split())
    return v if v else None

def dedupe_items(items):
    seen = set()
    result = []

    for item in items:
        row = {
            "name": norm(item.get("name")),
            "qty": norm(item.get("qty")),
            "unit_price": norm(item.get("unit_price")),
            "amount": norm(item.get("amount"))
        }

        # 商品名も金額もない行は捨てる
        if row["name"] is None and row["amount"] is None:
            continue

        key = (
            row["name"],
            row["qty"],
            row["unit_price"],
            row["amount"]
        )

        if key in seen:
            continue

        seen.add(key)
        result.append(row)

    return result

def main():
    img = preprocess_image(IMAGE_PATH)

    # 長いレシートでも読みやすいように縦分割
    parts = split_vertical(img, band_height=1400, overlap=220)

    all_items = []

    for idx, top, bottom, part in parts:
        image_b64 = image_to_b64(part)
        data = call_model(image_b64)
        all_items.extend(data.get("items", []))

    final_data = {
        "items": dedupe_items(all_items)
    }

    with open("items_only.json", "w", encoding="utf-8") as f:
        json.dump(final_data, f, ensure_ascii=False, indent=2)

    print(json.dumps(final_data, ensure_ascii=False, indent=2))

if __name__ == "__main__":
    main()