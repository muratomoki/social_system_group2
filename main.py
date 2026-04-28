import base64
import json
import requests
from io import BytesIO
from PIL import Image, ImageOps, ImageFilter
from fastapi import FastAPI, UploadFile, File, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from starlette.concurrency import run_in_threadpool

MODEL = "gemma4:e4b"
OLLAMA_URL = "http://localhost:11434/api/generate"
app = FastAPI()

# CORS設定
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 返してほしい形は「商品明細 + レシート全体の総額」
SCHEMA = {
    "type": "object",
    "properties": {
        "receipt_total": {"type": ["integer", "null"]},
        "receipt_date": {"type": ["integer", "null"]},
        "items": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "item": {"type": ["string", "null"]},
                    "num": {"type": ["integer", "null"]},
                    "amount": {"type": ["integer", "null"]},
                    "total": {"type": ["integer", "null"]},
                    "date": {"type": ["integer", "null"]},
                    "ingredients": {"type": ["integer", "null"]}
                },
                "required": ["item", "num", "amount", "total", "date", "ingredients"]
            }
        }
    },
    "required": ["receipt_total", "receipt_date", "items"]
}

# JSON形式の例
"""
    {
    "receipt_total": 1234,
    "receipt_date": 20260428,
    "items": [
        {
        "item": "商品名",
        "num": 1,
        "amount": 100,
        "total": 100,
        "date": 20260428,
        "ingredients": 1
        }
    ]
    }
"""

PROMPT = """
この画像はレシートの一部または全体です。
店ごとに形は違いますが、購入した商品・サービスの明細行だけを抽出してください。

必ず指定のJSONだけを返してください。
説明文は不要です。

トップレベルには receipt_total, receipt_date, items を入れてください。
receipt_total はレシート全体の支払総額を整数で入れてください。
receipt_date は購入日を YYYYMMDD の整数で入れてください。
読めない場合は null にしてください。

items には「商品名がある購入行」だけを入れてください。
次の情報は items に入れないでください。
店名、住所、電話番号、日時、レジ番号、担当者、会員情報、
小計、税、合計、預り、釣り、支払方法、カード、現金、ポイント、注意書き。

各商品は item, num, amount, total, date, ingredients だけを返してください。
item は商品名を入れてください。
num は個数を整数で入れてください。数量が明記されていない場合は 1 にしてください。
amount は1個あたりの金額、つまり単価を整数で入れてください。
total はその商品行の総額を整数で入れてください。
単価と総額の片方しか読めない場合は、読めた方だけ入れて、読めない方は null にしてください。
date は購入日を YYYYMMDD の整数で入れてください。読めない場合は receipt_date と同じ値にしてください。
ingredients は食材なら 1、食材でなければ 0、不明なら null にしてください。

値引き専用行、クーポン行、ポイント行は商品ではないので items に入れないでください。
読めない値は null にしてください。
推測で補完しないでください。
同じ商品が別の画像にも見えても、そのまま返してください。
"""

# 画像の前処理
def preprocess_image(img: Image.Image) -> Image.Image:
    # 白黒画像化
    img = img.convert("L")
    # 明暗差の自動調整
    img = ImageOps.autocontrast(img)
    # 画像拡大
    w, h = img.size
    img = img.resize((w * 2, h * 2))
    # シャープ化
    img = img.filter(ImageFilter.SHARPEN)
    return img

# アップロード画像の読み込み
async def read_upload_image(file: UploadFile) -> Image.Image:
    # 画像かどうかの判定
    if not file.content_type or not file.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="画像ファイルをアップロードしてください")

    try:
        # (非同期処理)ファイル読み込み
        contents = await file.read()
        img = Image.open(BytesIO(contents))
        img.load()
        # 確認用
        print("ファイル名:", file.filename)
        print("タイプ:", file.content_type)
        print("サイズ:", len(contents))

        return img
    except Exception as exc:
        raise HTTPException(status_code=400, detail="画像ファイルを読み込めませんでした") from exc

# 長いレシートの分割処理(1400px, overlap:220px)
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

# 画像のBase64化
def image_to_b64(img: Image.Image) -> str:
    buf = BytesIO()
    # PNG形式で保存
    img.save(buf, format="PNG")
    # Base64文字列に変換
    return base64.b64encode(buf.getvalue()).decode("utf-8")

# Ollama呼び出し
def call_model(image_b64: str) -> dict:
    payload = {
        "model": MODEL, # モデル名
        "prompt": PROMPT,   # プロンプト
        "images": [image_b64],  # Base64画像配列
        "format": SCHEMA,   # JSON形式指定
        "stream": False,    # 返答はまとめて
        "think": False  # 思考過程なし
    }

    # OllamaへのPOSTリクエスト
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

# 文字列や記号混じりの金額を整数に変換
def to_int(v, default=None):
    if v is None:
        return default

    if isinstance(v, int):
        return v

    text = str(v).strip()
    if not text:
        return default

    digits = "".join(ch for ch in text if ch.isdigit())
    if not digits:
        return default

    return int(digits)


def normalize_item(item, receipt_date=None):
    num = to_int(item.get("num") or item.get("qty"), 1)
    amount = to_int(item.get("amount") or item.get("unit_price"))
    total = to_int(item.get("total"))

    if total is None and num is not None and amount is not None:
        total = num * amount

    if amount is None and num not in (None, 0) and total is not None and total % num == 0:
        amount = total // num

    return {
        "item": norm(item.get("item") or item.get("name")),
        "num": num,
        "amount": amount,
        "total": total,
        "date": to_int(item.get("date"), receipt_date),
        "ingredients": to_int(item.get("ingredients"))
    }

# 重複した商品行の除去
def dedupe_items(items, receipt_date=None):
    seen = set()
    result = []

    for item in items:
        row = normalize_item(item, receipt_date)

        # 商品名も金額もない行は捨てる
        if row["item"] is None and row["total"] is None:
            continue

        key = (
            row["item"],
            row["num"],
            row["amount"],
            row["total"],
            row["date"],
            row["ingredients"]
        )

        if key in seen:
            continue

        seen.add(key)
        result.append(row)

    return result

def choose_receipt_value(values):
    for value in reversed(values):
        if value is not None:
            return value

    return None

# 商品明細の抽出(OCR部分)
def extract_items_from_image(img: Image.Image) -> dict:
    # 画像の補正
    img = preprocess_image(img)

    # 長いレシートでも読みやすいように縦分割
    parts = split_vertical(img, band_height=1400, overlap=220)

    all_items = []
    receipt_totals = []
    receipt_dates = []

    # AIへ分割画像を送信
    for idx, top, bottom, part in parts:
        image_b64 = image_to_b64(part)
        data = call_model(image_b64)
        all_items.extend(data.get("items", []))
        receipt_totals.append(to_int(data.get("receipt_total")))
        receipt_dates.append(to_int(data.get("receipt_date")))

    receipt_total = choose_receipt_value(receipt_totals)
    receipt_date = choose_receipt_value(receipt_dates)

    # 重複を除去した商品明細とレシート全体の総額を返却
    return {
        "receipt_total": receipt_total,
        "items": dedupe_items(all_items, receipt_date)
    }

# APIエンドポイント
@app.post("/receipt/items") # POST/receipt/items
async def receipt_items(file: UploadFile = File(...)):
    # アップロード画像の読み込み
    img = await read_upload_image(file)
    # 別スレッドでOCR処理の実行
    return await run_in_threadpool(extract_items_from_image, img)

# 起動処理
def main():
    import uvicorn

    # FastAPIを動かす(appをポート8000で起動)
    uvicorn.run("main:app", host="0.0.0.0", port=8000)

if __name__ == "__main__":
    main()
