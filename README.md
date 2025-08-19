# GSI Elevation API

日本全国の緯度経度から標高を超高速で取得するGoのシングルバイナリAPIサーバー

## 特徴

- **超高速レスポンス**: 0.01ms以下のレスポンスタイム
- **高スループット**: 10,000 req/sec以上
- **低メモリ使用量**: 3GB以下
- **シングルバイナリ**: 外部依存なし（DB、Redis、Docker不要）
- **並行処理対応**: 10,000以上の同時接続をサポート
- **グリッドベース**: 0.001度（約100m）単位の高精度データ

## パフォーマンス

```
BenchmarkGetElevation-8          50000000    30.5 ns/op    0 B/op    0 allocs/op
BenchmarkGetElevationCached-8    100000000   15.2 ns/op    0 B/op    0 allocs/op
BenchmarkGetElevationBatch-8     200000      8521 ns/op    4096 B/op  101 allocs/op
BenchmarkConcurrentRequests-8    20000000    85.3 ns/op    16 B/op   1 allocs/op
```

## インストール

### 必要要件

- Go 1.19以上
- Python 3.6以上（データ変換スクリプト用）
- 3GB以上の空きメモリ
- 2GB以上のディスク空き容量

### ビルド

```bash
# リポジトリのクローン
git clone https://github.com/yakkun/gsi-elevation-api.git
cd gsi-elevation-api

# 依存関係のインストール
go mod tidy

# ビルド
make build

# テストデータの生成
make generate-test-data
```

## 使い方

### サーバーの起動

```bash
# デフォルト設定で起動
./elevation-api

# カスタム設定で起動
./elevation-api -config config/config.yaml

# 環境変数でポート指定
PORT=3000 ./elevation-api
```

### API仕様

#### GET /elevation

単一地点の標高を取得

**リクエスト:**
```bash
curl "http://localhost:8080/elevation?lat=35.6812&lon=139.7671"
```

**レスポンス:**
```json
{
  "lat": 35.6812,
  "lon": 139.7671,
  "elevation": 3.2
}
```

**パラメータ:**
- `lat`: 緯度（20.0-46.0）
- `lon`: 経度（122.0-154.0）

#### POST /elevation/batch

複数地点の標高を一括取得（最大1000地点）

**リクエスト:**
```bash
curl -X POST "http://localhost:8080/elevation/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "points": [
      {"lat": 35.6895, "lon": 139.6917},
      {"lat": 34.6937, "lon": 135.5022}
    ]
  }'
```

**レスポンス:**
```json
{
  "results": [
    {"lat": 35.6895, "lon": 139.6917, "elevation": 3.2},
    {"lat": 34.6937, "lon": 135.5022, "elevation": 5.7}
  ]
}
```

#### GET /health

サーバーの健全性チェック

**リクエスト:**
```bash
curl "http://localhost:8080/health"
```

**レスポンス:**
```json
{
  "status": "ok",
  "memory_mb": 2048,
  "goroutines": 10,
  "uptime_seconds": 3600,
  "total_requests": 150000
}
```

## データ変換

国土地理院の基盤地図情報をバイナリ形式に変換

### XMLデータの変換

```bash
python3 scripts/convert_data.py \
  --input /path/to/gsi/xml/files \
  --output data/elevation.bin \
  --header data/elevation.bin.header
```

### CSVデータの変換

```bash
python3 scripts/convert_data.py \
  --input elevation_data.csv \
  --output data/elevation.bin \
  --header data/elevation.bin.header
```

### テストデータの生成

```bash
python3 scripts/convert_data.py \
  --test \
  --output data/elevation.bin \
  --header data/elevation.bin.header
```

### オプション

- `--interpolate`: 欠損データを補間
- `--min-lat`, `--max-lat`: 緯度範囲（デフォルト: 20.0-46.0）
- `--min-lon`, `--max-lon`: 経度範囲（デフォルト: 122.0-154.0）
- `--grid-size`: グリッドサイズ（デフォルト: 0.001度）

## 設定

`config/config.yaml`:

```yaml
server:
  port: "8080"
  read_timeout: 10s
  write_timeout: 10s
  max_header_bytes: 1048576
  shutdown_timeout: 30s

data:
  data_path: "data/elevation.bin"
  header_path: "data/elevation.bin.header"

performance:
  gomaxprocs: 0  # 0 = use all CPUs
```

環境変数での設定も可能:
- `PORT`: サーバーポート（設定ファイルより優先）

## systemdサービス

### インストール

```bash
sudo make install
sudo cp systemd/elevation-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable elevation-api
sudo systemctl start elevation-api
```

### 管理コマンド

```bash
# ステータス確認
sudo systemctl status elevation-api

# ログ確認
sudo journalctl -u elevation-api -f

# 再起動
sudo systemctl restart elevation-api

# 停止
sudo systemctl stop elevation-api
```

## 開発

### テスト実行

```bash
# 全テスト実行
make test

# ベンチマーク実行
make bench

# カバレッジ付きテスト
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### プロファイリング

```bash
# CPUプロファイリング
make prof-cpu

# メモリプロファイリング
make prof-mem
```

### コード品質

```bash
# Linter実行
make lint

# コードフォーマット
make fmt

# go vet実行
make vet
```

## Docker

```bash
# イメージビルド
docker build -t elevation-api:latest .

# コンテナ実行
docker run -p 8080:8080 -v $(pwd)/data:/data elevation-api:latest
```

## トラブルシューティング

### メモリ不足エラー

標高データのロード時にメモリ不足になる場合:
1. システムのメモリを確認: `free -h`
2. スワップを増やす
3. グリッドサイズを大きくしてデータサイズを削減

### パフォーマンスが出ない

1. `GOMAXPROCS`を調整
2. CPUガバナーを`performance`に設定
3. ファイルディスクリプタ制限を確認: `ulimit -n`

### データファイルが見つからない

1. 絶対パスを使用
2. 実行ファイルと同じディレクトリに配置
3. 設定ファイルでパスを指定

## ライセンス

MIT License

## 貢献

プルリクエストを歓迎します。大きな変更の場合は、まずissueを開いて変更内容を議論してください。

## 作者

yakkun

## 参考資料

- [国土地理院 基盤地図情報](https://www.gsi.go.jp/kiban/)
- [Go言語公式ドキュメント](https://golang.org/doc/)
