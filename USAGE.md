## 設定

### config.yamlの編集

監視対象サイトとメール設定を編集します：

```bash
vi config.yaml
```

#### 主要な設定項目：

**1. 監視対象サイト**
```yaml
sites:
  - url: your-site.example.com
    port: 443
    name: "本番サイト"
  - url: api.example.com
    port: 443
    name: "API サーバー"
```

**2. アラートしきい値**
```yaml
alert:
  warning_days: 30  # 30日以内で警告
  critical_days: 7  # 7日以内で緊急警告
```

**3. メール設定**

**SSL接続を使用する場合（ポート465）：**
```yaml
email:
  enabled: true
  smtp:
    host: "smtp.example.com"
    port: 465
    use_ssl: true  # SSL接続を使用
    use_tls: false  # TLSは使用しない
    username: "your-email@example.com"
    password: "your-password"
  from: "cert-checker@example.com"
  to:
    - "admin@example.com"
  subject: "SSL証明書有効期限チェック結果"
```

**TLS接続（STARTTLS）を使用する場合（ポート587、Gmailなど）：**
```yaml
email:
  enabled: true
  smtp:
    host: "smtp.gmail.com"
    port: 587
    use_ssl: false
    use_tls: true  # STARTTLSを使用
    username: "your-email@gmail.com"
    password: "your-app-password"
  from: "cert-checker@example.com"
  to:
    - "admin@example.com"
  subject: "SSL証明書有効期限チェック結果"
```

**4. Discord通知設定**

Discord Webhookを使用して通知を受け取ることができます：

```yaml
discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
  notify_on:
    - "WARNING"
    - "CRITICAL"
    - "ERROR"
```

**Discord Webhook URLの取得方法：**
1. Discordサーバーの設定を開く
2. 「連携サービス」→「ウェブフック」を選択
3. 「新しいウェブフック」をクリック
4. ウェブフック名を設定（例：SSL証明書チェッカー）
5. 通知を送信したいチャンネルを選択
6. 「ウェブフックURLをコピー」をクリック
7. コピーしたURLをconfig.yamlに設定

**通知条件の設定：**
- `notify_on`: 通知するステータスを指定
  - `OK`: 正常な証明書
  - `WARNING`: 警告（デフォルト: 30日以内）
  - `CRITICAL`: 緊急（デフォルト: 7日以内）
  - `ERROR`: エラー（証明書取得失敗など）
- 空配列またはこのフィールドを削除すると、全てのステータスで通知

**通知の見た目：**
- Discordにはリッチな埋め込みメッセージとして表示
- ステータスに応じて色分け（緑=OK、オレンジ=警告、赤=緊急）
- 各サイトの証明書情報を個別のカードで表示

## 実行方法

### コマンドラインオプション

```bash
./cert-checker [オプション]

オプション:
  -config string
        設定ファイルのパス (デフォルト: "config.yaml")
```

### 手動実行

#### 通常実行（デフォルト設定ファイル）
```bash
./cert-checker
```

#### カスタム設定ファイルを指定
```bash
./cert-checker -config /path/to/custom-config.yaml
```

実行結果はコンソールに表示され、設定に応じてメールも送信されます。

#### メール送信を無効にしてテスト実行
config.yamlで`email.enabled`を`false`に設定してから実行

### 定期実行（cron）

#### cronの設定
```bash
crontab -e
```

以下を追加：

**毎日午前9時に実行（デフォルト設定ファイル）：**
```cron
0 9 * * * cd /var/tmp/cert_checker_go && ./cert-checker >> /var/log/cert_checker.log 2>&1
```

**毎週月曜日午前9時に実行（カスタム設定ファイル）：**
```cron
0 9 * * 1 cd /var/tmp/cert_checker_go && ./cert-checker -config /path/to/custom-config.yaml >> /var/log/cert_checker.log 2>&1
```

**毎月1日午前9時に実行：**
```cron
0 9 1 * * cd /var/tmp/cert_checker_go && ./cert-checker >> /var/log/cert_checker.log 2>&1
```

**複数の設定ファイルで異なるスケジュールで実行：**
```cron
# 本番環境: 毎日午前9時
0 9 * * * cd /var/tmp/cert_checker_go && ./cert-checker -config production.yaml >> /var/log/cert_checker_prod.log 2>&1

# 開発環境: 毎週月曜日午前10時
0 10 * * 1 cd /var/tmp/cert_checker_go && ./cert-checker -config development.yaml >> /var/log/cert_checker_dev.log 2>&1
```

## 出力例

### コンソール出力
```
================================================================================
SSL証明書有効期限チェック結果
チェック日時: 2025-12-01 18:03:54
================================================================================

サイト名: Google
URL: www.google.com:443
ステータス: OK
発行者: Google Trust Services
主体者: www.google.com
有効期限開始: 2025-10-27 08:35:45
有効期限終了: 2026-01-19 08:35:44
残り日数: 48日
--------------------------------------------------------------------------------
```

### ログファイル
```
2025/12/01 18:03:53 SSL証明書チェッカーを開始します
2025/12/01 18:03:53 3サイトのチェックを開始します
2025/12/01 18:03:53 チェック開始: Google (www.google.com:443)
2025/12/01 18:03:54 チェック開始: GitHub (www.github.com:443)
2025/12/01 18:03:54 チェック開始: Example Site (www.example.com:443)
2025/12/01 18:03:54 すべてのサイトのチェックが完了しました
2025/12/01 18:03:55 メールを送信しました
2025/12/01 18:03:55 SSL証明書チェッカーを終了します
```

### メール
- **件名**: SSL証明書有効期限チェック結果
- **本文**: HTML形式の見やすい表形式レポート
  - 色分けされたステータス（緑=OK、オレンジ=警告、赤=緊急）
  - 各サイトの証明書情報
  - 残り日数

## トラブルシューティング

### よくある問題

**1. "証明書の取得に失敗" エラー**
- ネットワーク接続を確認
- ファイアウォールの設定を確認
- URLとポート番号が正しいか確認

**2. メール送信エラー**
- SMTPサーバー設定を確認
- 認証情報（ユーザー名・パスワード）を確認
- ファイアウォールでSMTPポート（465または587）が開いているか確認

**3. "設定ファイルの読み込みに失敗"**
- config.yamlが同じディレクトリにあるか確認
- YAMLの文法が正しいか確認（インデントなど）

**4. ビルドエラー**
```bash
# Go のバージョン確認
go version

# 依存関係の再取得
go mod tidy

# クリーンビルド
go clean
go build -buildvcs=false -o cert-checker
```

### ログの確認
```bash
# アプリケーションログの確認
cat cert_checker.log

# cronログの確認
tail -f /var/log/cert_checker.log
```

## セキュリティに関する注意

1. **認証情報の保護**
   - config.yamlにパスワードを平文で保存することになるため、ファイルのパーミッションを適切に設定
   ```bash
   chmod 600 config.yaml
   ```

2. **バイナリの配置**
   - 実行ファイルは適切なディレクトリに配置し、必要最小限の権限で実行
