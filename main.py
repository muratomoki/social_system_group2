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

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

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
        print("サイズ:", len(img))

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

# 重複した商品行の除去
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

# 商品明細の抽出(OCR部分)
def extract_items_from_image(img: Image.Image) -> dict:
    # 画像の補正
    img = preprocess_image(img)

    # 長いレシートでも読みやすいように縦分割
    parts = split_vertical(img, band_height=1400, overlap=220)

    all_items = []

    # AIへ分割画像を送信
    for idx, top, bottom, part in parts:
        image_b64 = image_to_b64(part)
        data = call_model(image_b64)
        all_items.extend(data.get("items", []))

    # 重複を除去した商品明細を返却
    return {
        "items": dedupe_items(all_items)
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
