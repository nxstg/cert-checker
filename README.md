# SSL証明書有効期限チェッカー

[![CI](https://github.com/nxstg/cert-checker/workflows/CI/badge.svg)](https://github.com/nxstg/cert-checker/actions)
[![Release](https://github.com/nxstg/cert-checker/workflows/Release/badge.svg)](https://github.com/nxstg/cert-checker/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

複数のサイトのSSL証明書の有効期限を確認し、結果をメールで送信するツールです。

## 機能

- 複数のWebサイトのSSL証明書の有効期限を確認
- 有効期限までの残り日数を計算
- 警告しきい値を設定可能（例：30日以内に期限切れの場合は警告）
- 結果をメールで送信(マルチパートメッセージとして送信)
- Discord通知に対応（Webhook経由）
- 証明書が期限切れの場合、終了コード1を返却

## インストール

### オプション1: ビルド済みバイナリを使用（推奨）

[GitHub Releases](https://github.com/nxstg/cert-checker/releases)から最新版をダウンロード：

```bash
# Linux (x64) の例
wget https://github.com/nxstg/cert-checker/releases/latest/download/cert-checker-linux-amd64
chmod +x cert-checker-linux-amd64
mv cert-checker-linux-amd64 cert-checker

# 設定ファイルをダウンロード
wget https://github.com/nxstg/cert-checker/releases/latest/download/config.yaml.example
cp config.yaml.example config.yaml
vi config.yaml
```

**利用可能なバイナリ:**
- `cert-checker-linux-amd64` - Linux (x64)
- `cert-checker-linux-arm64` - Linux (ARM64)
- `cert-checker-darwin-amd64` - macOS (Intel)
- `cert-checker-darwin-arm64` - macOS (Apple Silicon)
- `cert-checker-windows-amd64.exe` - Windows

### オプション2: ソースからビルド

```bash
# リポジトリをクローン
git clone https://github.com/nxstg/cert-checker.git
cd cert-checker

# 依存パッケージのダウンロード
go mod tidy

# ビルド
go build -buildvcs=false -o cert-checker
```

## クイックスタート

### 1. 設定ファイルの準備
```bash
# サンプル設定ファイルをコピー
cp config.yaml.example config.yaml

# 設定ファイルを編集（監視対象サイト、メール設定など）
vi config.yaml
```

### 2. ビルドと実行
```bash
# 依存パッケージのダウンロード
go mod tidy

# ビルド
# Linux/Mac
go build -buildvcs=false -o cert-checker

# Windows
go build -buildvcs=false -o cert-checker.exe

# クロスコンパイル例（Linuxで実行するバイナリをMacで作成）
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o cert-checker-linux

# 実行（デフォルトでconfig.yamlを使用）
./cert-checker

# カスタム設定ファイルを指定
./cert-checker -config /path/to/custom-config.yaml
```

## コマンドラインオプション

```bash
./cert-checker [オプション]

オプション:
  -config string
        設定ファイルのパス (デフォルト: "config.yaml")
```

## 構成ファイル

- `main.go` - メインソースコード
- `config.yaml.example` - 設定ファイルのサンプル
- `config.yaml` - 実際の設定ファイル（各自で作成、Gitには含まれません）
- `go.mod` - Go モジュール定義


### 3. cronで定期実行（例：毎週月曜日9時）
```bash
# デフォルト設定ファイルを使用
0 9 * * 1 cd /path/to/cert_checker_go && ./cert-checker

# カスタム設定ファイルを使用
0 9 * * 1 cd /path/to/cert_checker_go && ./cert-checker -config /path/to/custom-config.yaml
```
## システム要件

- Go 1.21以上（ビルド時のみ）
- 実行時は依存関係なし

## 詳細な使用方法

詳細はUSAGE.mdを参照してください。
