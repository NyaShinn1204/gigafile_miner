<h1 align="center">
  ギガファイル便総当たりツール
</h1>

<h3 align="center">
  総当たりを行いファイルをダウンロードするツールです
</h3>

## ⚠️IMPORTANT

**このツールでダウンロードしたファイルに関しては一切責任を負いません**

**自己責任でファイルをダウンロードしてください**

## Installation

**※ Go 1.22.3 以上のバージョンが必要です。**

**※ http プロキシが必須です (IP:PORT形式のみ現在サポート)**

依存関係はgo runすればインストールしてくれます。

> [!TIP]
> 開発バージョンです! 正式リリースはありません

```bash
git clone https://github.com/NyaShinn1204/gigafile_miner.git

go run main.go
```

## コンフィグについて

デフォルトには毎秒25スレッドとなっています。

変更したいときは、main.go内の265行目
```go
～～～～～
264: baseURL := "https://xgf.nu/"
265: numWorkers := 25 // 並行して動作させるワーカーの数       ここ！！
266: var wg sync.WaitGroup
～～～～～
```
の25を好きな数字に変えてください
