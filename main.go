package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 設定ファイルの構造
type Config struct {
	Sites []Site `yaml:"sites"`
	Alert struct {
		WarningDays  int `yaml:"warning_days"`
		CriticalDays int `yaml:"critical_days"`
	} `yaml:"alert"`
	Email struct {
		Enabled bool `yaml:"enabled"`
		SMTP    struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			UseSSL   bool   `yaml:"use_ssl"`
			UseTLS   bool   `yaml:"use_tls"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"smtp"`
		From    string   `yaml:"from"`
		To      []string `yaml:"to"`
		Subject string   `yaml:"subject"`
	} `yaml:"email"`
	Discord struct {
		Enabled    bool     `yaml:"enabled"`
		WebhookURL string   `yaml:"webhook_url"`
		NotifyOn   []string `yaml:"notify_on"`
	} `yaml:"discord"`
	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`
}

// Site 監視対象サイト
type Site struct {
	URL  string `yaml:"url"`
	Port int    `yaml:"port"`
	Name string `yaml:"name"`
}

// CertInfo 証明書情報
type CertInfo struct {
	SiteName      string
	URL           string
	Port          int
	Issuer        string
	Subject       string
	NotBefore     time.Time
	NotAfter      time.Time
	DaysRemaining int
	Status        string // OK, WARNING, CRITICAL, ERROR
	ErrorMessage  string
}

// Logger ロガー
var Logger *log.Logger

// JSTタイムゾーン
var JST *time.Location

func init() {
	// JSTタイムゾーンを設定
	var err error
	JST, err = time.LoadLocation("Asia/Tokyo")
	if err != nil {
		// タイムゾーンの読み込みに失敗した場合はUTC+9で設定
		JST = time.FixedZone("Asia/Tokyo", 9*60*60)
	}
}

func main() {
	// コマンドライン引数の解析
	configPath := flag.String("config", "config.yaml", "設定ファイルのパス")
	flag.Parse()

	// 設定ファイルの読み込み
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	// ロガーのセットアップ
	setupLogger(config)

	Logger.Println("SSL証明書チェッカーを開始します")

	// 証明書チェック
	results := checkAllSites(config)

	// レポート生成
	textReport := generateTextReport(results)
	fmt.Println("\n" + textReport)

	// メール送信
	if config.Email.Enabled {
		if err := sendEmail(config, results); err != nil {
			Logger.Printf("メール送信に失敗しました: %v", err)
		} else {
			Logger.Println("メールを送信しました")
		}
	} else {
		Logger.Println("メール送信は無効です")
	}

	// Discord通知
	if err := sendDiscordNotification(config, results); err != nil {
		Logger.Printf("Discord通知でエラーが発生しました: %v", err)
	}

	Logger.Println("SSL証明書チェッカーを終了します")

	// 問題があるサイトがある場合は終了コード1
	hasIssues := false
	for _, result := range results {
		if result.Status == "WARNING" || result.Status == "CRITICAL" || result.Status == "ERROR" {
			hasIssues = true
			break
		}
	}
	if hasIssues {
		os.Exit(1)
	}
}

// loadConfig 設定ファイルを読み込む
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setupLogger ロガーをセットアップ
func setupLogger(config *Config) {
	var output *os.File
	if config.Logging.File != "" {
		f, err := os.OpenFile(config.Logging.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("ログファイルのオープンに失敗: %v", err)
			output = os.Stdout
		} else {
			output = f
		}
	} else {
		output = os.Stdout
	}

	Logger = log.New(output, "", log.LstdFlags)
}

// checkAllSites すべてのサイトをチェック
func checkAllSites(config *Config) []CertInfo {
	Logger.Printf("%dサイトのチェックを開始します", len(config.Sites))

	results := make([]CertInfo, 0, len(config.Sites))
	for _, site := range config.Sites {
		result := checkCertificate(config, site)
		results = append(results, result)
	}

	Logger.Println("すべてのサイトのチェックが完了しました")
	return results
}

// checkCertificate 証明書をチェック
func checkCertificate(config *Config, site Site) CertInfo {
	Logger.Printf("チェック開始: %s (%s:%d)", site.Name, site.URL, site.Port)

	// デフォルトポート
	if site.Port == 0 {
		site.Port = 443
	}
	if site.Name == "" {
		site.Name = site.URL
	}

	// 証明書取得
	conf := &tls.Config{
		ServerName: site.URL,
	}

	address := fmt.Sprintf("%s:%d", site.URL, site.Port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, conf)
	if err != nil {
		errorMsg := fmt.Sprintf("証明書の取得に失敗: %v", err)
		Logger.Printf("%s:%d - %s", site.URL, site.Port, errorMsg)
		return CertInfo{
			SiteName:     site.Name,
			URL:          site.URL,
			Port:         site.Port,
			Status:       "ERROR",
			ErrorMessage: errorMsg,
		}
	}
	defer conn.Close()

	// 証明書情報の取得
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return CertInfo{
			SiteName:     site.Name,
			URL:          site.URL,
			Port:         site.Port,
			Status:       "ERROR",
			ErrorMessage: "証明書が見つかりません",
		}
	}

	cert := certs[0]

	// 残り日数を計算
	now := time.Now()
	daysRemaining := int(cert.NotAfter.Sub(now).Hours() / 24)

	// ステータスの判定
	var status string
	if daysRemaining < 0 {
		status = "CRITICAL"
	} else if daysRemaining <= config.Alert.CriticalDays {
		status = "CRITICAL"
	} else if daysRemaining <= config.Alert.WarningDays {
		status = "WARNING"
	} else {
		status = "OK"
	}

	// 発行者情報
	issuer := cert.Issuer.Organization
	if len(issuer) == 0 {
		issuer = []string{cert.Issuer.CommonName}
	}
	issuerStr := strings.Join(issuer, ", ")
	if issuerStr == "" {
		issuerStr = "Unknown"
	}

	return CertInfo{
		SiteName:      site.Name,
		URL:           site.URL,
		Port:          site.Port,
		Issuer:        issuerStr,
		Subject:       cert.Subject.CommonName,
		NotBefore:     cert.NotBefore,
		NotAfter:      cert.NotAfter,
		DaysRemaining: daysRemaining,
		Status:        status,
	}
}

// generateTextReport テキストレポートを生成
func generateTextReport(results []CertInfo) string {
	var sb strings.Builder

	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("SSL証明書有効期限チェック結果\n")
	sb.WriteString(fmt.Sprintf("チェック日時: %s\n", time.Now().In(JST).Format("2006-01-02 15:04:05")))
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	for _, cert := range results {
		sb.WriteString(fmt.Sprintf("サイト名: %s\n", cert.SiteName))
		sb.WriteString(fmt.Sprintf("URL: %s:%d\n", cert.URL, cert.Port))
		sb.WriteString(fmt.Sprintf("ステータス: %s\n", cert.Status))

		if cert.Status != "ERROR" {
			sb.WriteString(fmt.Sprintf("発行者: %s\n", cert.Issuer))
			sb.WriteString(fmt.Sprintf("主体者: %s\n", cert.Subject))
			sb.WriteString(fmt.Sprintf("有効期限開始: %s JST\n", cert.NotBefore.In(JST).Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("有効期限終了: %s JST\n", cert.NotAfter.In(JST).Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("残り日数: %d日\n", cert.DaysRemaining))
		} else {
			sb.WriteString(fmt.Sprintf("エラー: %s\n", cert.ErrorMessage))
		}

		sb.WriteString(strings.Repeat("-", 80) + "\n")
	}

	return sb.String()
}

// generateHTMLReport HTMLレポートを生成
func generateHTMLReport(results []CertInfo) string {
	checkTime := time.Now().In(JST).Format("2006-01-02 15:04:05")

	html := fmt.Sprintf(`<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%%; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        tr:nth-child(even) { background-color: #f2f2f2; }
        .ok { color: green; font-weight: bold; }
        .warning { color: orange; font-weight: bold; }
        .critical { color: red; font-weight: bold; }
        .error { color: darkred; font-weight: bold; }
    </style>
</head>
<body>
    <h1>SSL証明書有効期限チェック結果</h1>
    <p>チェック日時: %s</p>
    <table>
        <tr>
            <th>サイト名</th>
            <th>URL</th>
            <th>発行者</th>
            <th>有効期限</th>
            <th>残り日数</th>
            <th>ステータス</th>
        </tr>
`, checkTime)

	for _, cert := range results {
		statusClass := strings.ToLower(cert.Status)

		if cert.Status != "ERROR" {
			html += fmt.Sprintf(`        <tr>
            <td>%s</td>
            <td>%s:%d</td>
            <td>%s</td>
            <td>%s JST</td>
            <td>%d日</td>
            <td class="%s">%s</td>
        </tr>
`, cert.SiteName, cert.URL, cert.Port, cert.Issuer,
				cert.NotAfter.In(JST).Format("2006-01-02"), cert.DaysRemaining,
				statusClass, cert.Status)
		} else {
			html += fmt.Sprintf(`        <tr>
            <td>%s</td>
            <td>%s:%d</td>
            <td colspan="3">%s</td>
            <td class="%s">%s</td>
        </tr>
`, cert.SiteName, cert.URL, cert.Port, cert.ErrorMessage, statusClass, cert.Status)
		}
	}

	html += `    </table>
</body>
</html>`

	return html
}

// sendEmail メールを送信
func sendEmail(config *Config, results []CertInfo) error {
	// メッセージの作成
	textReport := generateTextReport(results)
	htmlReport := generateHTMLReport(results)

	// マルチパートメッセージの作成
	boundary := "boundary123456789"
	message := fmt.Sprintf("From: %s\r\n", config.Email.From)
	message += fmt.Sprintf("To: %s\r\n", strings.Join(config.Email.To, ", "))
	message += fmt.Sprintf("Subject: %s\r\n", config.Email.Subject)
	message += "MIME-Version: 1.0\r\n"
	message += fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary)
	message += "\r\n"

	// テキストパート
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += textReport + "\r\n"

	// HTMLパート
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += htmlReport + "\r\n"

	message += fmt.Sprintf("--%s--\r\n", boundary)

	// SMTP接続
	smtpAddr := fmt.Sprintf("%s:%d", config.Email.SMTP.Host, config.Email.SMTP.Port)

	var auth smtp.Auth
	if config.Email.SMTP.Username != "" && config.Email.SMTP.Password != "" {
		auth = smtp.PlainAuth("", config.Email.SMTP.Username, config.Email.SMTP.Password, config.Email.SMTP.Host)
	}

	// SSL接続の場合
	if config.Email.SMTP.UseSSL {
		tlsConfig := &tls.Config{
			ServerName: config.Email.SMTP.Host,
		}

		conn, err := tls.Dial("tcp", smtpAddr, tlsConfig)
		if err != nil {
			return fmt.Errorf("SSL接続に失敗: %v", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, config.Email.SMTP.Host)
		if err != nil {
			return fmt.Errorf("SMTPクライアントの作成に失敗: %v", err)
		}
		defer client.Close()

		// 認証
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("認証に失敗: %v", err)
			}
		}

		// 送信
		if err := client.Mail(config.Email.From); err != nil {
			return fmt.Errorf("MAIL FROMに失敗: %v", err)
		}
		for _, to := range config.Email.To {
			if err := client.Rcpt(to); err != nil {
				return fmt.Errorf("RCPT TOに失敗: %v", err)
			}
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("DATAコマンドに失敗: %v", err)
		}
		if _, err := w.Write([]byte(message)); err != nil {
			return fmt.Errorf("メッセージの送信に失敗: %v", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("メッセージのクローズに失敗: %v", err)
		}

		return client.Quit()
	}

	// TLS接続（STARTTLS）の場合
	if config.Email.SMTP.UseTLS {
		return smtp.SendMail(smtpAddr, auth, config.Email.From, config.Email.To, []byte(message))
	}

	// 暗号化なしの場合
	return smtp.SendMail(smtpAddr, auth, config.Email.From, config.Email.To, []byte(message))
}

// sendDiscordNotification Discordに通知を送信
func sendDiscordNotification(config *Config, results []CertInfo) error {
	if !config.Discord.Enabled {
		Logger.Println("Discord通知は無効です")
		return nil
	}

	webhookURL := config.Discord.WebhookURL
	if webhookURL == "" || webhookURL == "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN" {
		Logger.Println("Discord Webhook URLが設定されていません")
		return nil
	}

	// 通知対象の結果をフィルタリング
	notifyOn := config.Discord.NotifyOn
	filteredResults := []CertInfo{}

	if len(notifyOn) > 0 {
		for _, result := range results {
			for _, status := range notifyOn {
				if result.Status == status {
					filteredResults = append(filteredResults, result)
					break
				}
			}
		}
	} else {
		filteredResults = results
	}

	if len(filteredResults) == 0 {
		Logger.Println("Discord通知対象の結果がありません")
		return nil
	}

	// Discord Embed形式でメッセージを作成
	type EmbedField struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Inline bool   `json:"inline"`
	}

	type Embed struct {
		Title     string       `json:"title"`
		Color     int          `json:"color"`
		Fields    []EmbedField `json:"fields"`
		Timestamp string       `json:"timestamp"`
	}

	type Payload struct {
		Username string  `json:"username"`
		Embeds   []Embed `json:"embeds"`
	}

	embeds := []Embed{}
	for _, cert := range filteredResults {
		// ステータスに応じた色を設定
		colorMap := map[string]int{
			"OK":       0x00FF00, // 緑
			"WARNING":  0xFFA500, // オレンジ
			"CRITICAL": 0xFF0000, // 赤
			"ERROR":    0x8B0000, // 暗い赤
		}
		color := colorMap[cert.Status]
		if color == 0 {
			color = 0x808080 // グレー
		}

		// Embedフィールドの作成
		fields := []EmbedField{}
		if cert.Status != "ERROR" {
			fields = []EmbedField{
				{Name: "URL", Value: fmt.Sprintf("%s:%d", cert.URL, cert.Port), Inline: true},
				{Name: "ステータス", Value: cert.Status, Inline: true},
				{Name: "残り日数", Value: fmt.Sprintf("%d日", cert.DaysRemaining), Inline: true},
				{Name: "発行者", Value: cert.Issuer, Inline: false},
				{Name: "有効期限", Value: fmt.Sprintf("%s JST", cert.NotAfter.In(JST).Format("2006-01-02 15:04:05")), Inline: false},
			}
		} else {
			fields = []EmbedField{
				{Name: "URL", Value: fmt.Sprintf("%s:%d", cert.URL, cert.Port), Inline: true},
				{Name: "ステータス", Value: cert.Status, Inline: true},
				{Name: "エラー", Value: cert.ErrorMessage, Inline: false},
			}
		}

		embed := Embed{
			Title:     fmt.Sprintf("🔒 %s", cert.SiteName),
			Color:     color,
			Fields:    fields,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		embeds = append(embeds, embed)
	}

	payload := Payload{
		Username: "SSL証明書チェッカー",
		Embeds:   embeds,
	}

	// JSONに変換
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSONのマーシャルに失敗: %v", err)
	}

	// Webhookに送信
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("Discord通知の送信に失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		Logger.Println("Discord通知を送信しました")
	} else {
		Logger.Printf("Discord通知の送信結果: %d", resp.StatusCode)
	}

	return nil
}
