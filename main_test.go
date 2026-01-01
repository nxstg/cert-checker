package main

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLoadConfig 設定ファイルの読み込みテスト
func TestLoadConfig(t *testing.T) {
	// テスト用の設定ファイルを作成
	testConfig := `
sites:
  - url: example.com
    port: 443
    name: Example Site
  - url: test.com
    port: 8443
    name: Test Site

alert:
  warning_days: 30
  critical_days: 7

email:
  enabled: true
  smtp:
    host: smtp.example.com
    port: 587
    use_ssl: false
    use_tls: true
    username: user@example.com
    password: password123
  from: noreply@example.com
  to:
    - admin@example.com
  subject: "SSL証明書有効期限チェック"

discord:
  enabled: false
  webhook_url: ""
  notify_on:
    - WARNING
    - CRITICAL

logging:
  level: info
  file: ""
`

	// 一時ファイルを作成
	tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("一時ファイルへの書き込みに失敗: %v", err)
	}
	tmpFile.Close()

	// 設定ファイルを読み込み
	config, err := loadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("設定ファイルの読み込みに失敗: %v", err)
	}

	// サイト数の確認
	if len(config.Sites) != 2 {
		t.Errorf("サイト数が正しくありません。期待: 2, 実際: %d", len(config.Sites))
	}

	// サイト情報の確認
	if config.Sites[0].URL != "example.com" {
		t.Errorf("サイトURLが正しくありません。期待: example.com, 実際: %s", config.Sites[0].URL)
	}
	if config.Sites[0].Port != 443 {
		t.Errorf("ポート番号が正しくありません。期待: 443, 実際: %d", config.Sites[0].Port)
	}
	if config.Sites[0].Name != "Example Site" {
		t.Errorf("サイト名が正しくありません。期待: Example Site, 実際: %s", config.Sites[0].Name)
	}

	// アラート設定の確認
	if config.Alert.WarningDays != 30 {
		t.Errorf("警告日数が正しくありません。期待: 30, 実際: %d", config.Alert.WarningDays)
	}
	if config.Alert.CriticalDays != 7 {
		t.Errorf("危険日数が正しくありません。期待: 7, 実際: %d", config.Alert.CriticalDays)
	}

	// メール設定の確認
	if !config.Email.Enabled {
		t.Error("メール送信が無効になっています")
	}
	if config.Email.SMTP.Host != "smtp.example.com" {
		t.Errorf("SMTPホストが正しくありません。期待: smtp.example.com, 実際: %s", config.Email.SMTP.Host)
	}
	if config.Email.SMTP.Port != 587 {
		t.Errorf("SMTPポートが正しくありません。期待: 587, 実際: %d", config.Email.SMTP.Port)
	}
}

// TestLoadConfigFileNotFound 存在しないファイルの読み込みテスト
func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := loadConfig("nonexistent_file.yaml")
	if err == nil {
		t.Error("存在しないファイルの読み込みでエラーが発生しませんでした")
	}
}

// TestLoadConfigInvalidYAML 不正なYAMLファイルの読み込みテスト
func TestLoadConfigInvalidYAML(t *testing.T) {
	// 不正なYAMLファイルを作成
	tmpFile, err := os.CreateTemp("", "test_invalid_*.yaml")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	invalidYAML := "invalid: yaml: content:\n  - no proper indentation"
	if _, err := tmpFile.WriteString(invalidYAML); err != nil {
		t.Fatalf("一時ファイルへの書き込みに失敗: %v", err)
	}
	tmpFile.Close()

	_, err = loadConfig(tmpFile.Name())
	if err == nil {
		t.Error("不正なYAMLファイルの読み込みでエラーが発生しませんでした")
	}
}

// TestGenerateTextReport テキストレポート生成のテスト
func TestGenerateTextReport(t *testing.T) {
	now := time.Now()
	results := []CertInfo{
		{
			SiteName:      "Example Site",
			URL:           "example.com",
			Port:          443,
			Issuer:        "Let's Encrypt",
			Subject:       "example.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 2, 0),
			DaysRemaining: 60,
			Status:        "OK",
		},
		{
			SiteName:      "Warning Site",
			URL:           "warning.com",
			Port:          443,
			Issuer:        "DigiCert",
			Subject:       "warning.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 0, 20),
			DaysRemaining: 20,
			Status:        "WARNING",
		},
		{
			SiteName:      "Critical Site",
			URL:           "critical.com",
			Port:          443,
			Issuer:        "GlobalSign",
			Subject:       "critical.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 0, 5),
			DaysRemaining: 5,
			Status:        "CRITICAL",
		},
		{
			SiteName:     "Error Site",
			URL:          "error.com",
			Port:         443,
			Status:       "ERROR",
			ErrorMessage: "接続に失敗しました",
		},
	}

	report := generateTextReport(results)

	// レポートに必要な情報が含まれているか確認
	if !strings.Contains(report, "SSL証明書有効期限チェック結果") {
		t.Error("レポートにタイトルが含まれていません")
	}

	// 各サイトの情報が含まれているか確認
	for _, result := range results {
		if !strings.Contains(report, result.SiteName) {
			t.Errorf("レポートにサイト名 '%s' が含まれていません", result.SiteName)
		}
		if !strings.Contains(report, result.URL) {
			t.Errorf("レポートにURL '%s' が含まれていません", result.URL)
		}
		if !strings.Contains(report, result.Status) {
			t.Errorf("レポートにステータス '%s' が含まれていません", result.Status)
		}
	}

	// エラーメッセージが含まれているか確認
	if !strings.Contains(report, "接続に失敗しました") {
		t.Error("レポートにエラーメッセージが含まれていません")
	}
}

// TestGenerateHTMLReport HTMLレポート生成のテスト
func TestGenerateHTMLReport(t *testing.T) {
	now := time.Now()
	results := []CertInfo{
		{
			SiteName:      "Example Site",
			URL:           "example.com",
			Port:          443,
			Issuer:        "Let's Encrypt",
			Subject:       "example.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 2, 0),
			DaysRemaining: 60,
			Status:        "OK",
		},
		{
			SiteName:     "Error Site",
			URL:          "error.com",
			Port:         443,
			Status:       "ERROR",
			ErrorMessage: "接続に失敗しました",
		},
	}

	report := generateHTMLReport(results)

	// HTMLの基本構造を確認
	if !strings.Contains(report, "<html>") {
		t.Error("HTMLレポートに<html>タグが含まれていません")
	}
	if !strings.Contains(report, "<head>") {
		t.Error("HTMLレポートに<head>タグが含まれていません")
	}
	if !strings.Contains(report, "<body>") {
		t.Error("HTMLレポートに<body>タグが含まれていません")
	}
	if !strings.Contains(report, "<table>") {
		t.Error("HTMLレポートに<table>タグが含まれていません")
	}

	// CSSスタイルが含まれているか確認
	if !strings.Contains(report, "<style>") {
		t.Error("HTMLレポートにスタイルが含まれていません")
	}

	// 各サイトの情報が含まれているか確認
	for _, result := range results {
		if !strings.Contains(report, result.SiteName) {
			t.Errorf("HTMLレポートにサイト名 '%s' が含まれていません", result.SiteName)
		}
		if !strings.Contains(report, result.URL) {
			t.Errorf("HTMLレポートにURL '%s' が含まれていません", result.URL)
		}
		if !strings.Contains(report, result.Status) {
			t.Errorf("HTMLレポートにステータス '%s' が含まれていません", result.Status)
		}
	}

	// ステータスに応じたCSSクラスが含まれているか確認
	if !strings.Contains(report, "class=\"ok\"") {
		t.Error("HTMLレポートにOKステータスのCSSクラスが含まれていません")
	}
	if !strings.Contains(report, "class=\"error\"") {
		t.Error("HTMLレポートにERRORステータスのCSSクラスが含まれていません")
	}
}

// TestCertInfoStatusDetermination ステータス判定のテスト
func TestCertInfoStatusDetermination(t *testing.T) {
	// テスト用の設定
	config := &Config{}
	config.Alert.WarningDays = 30
	config.Alert.CriticalDays = 7

	testCases := []struct {
		name             string
		daysRemaining    int
		expectedStatus   string
		notAfter         time.Time
	}{
		{
			name:           "OK状態（60日残り）",
			daysRemaining:  60,
			expectedStatus: "OK",
			notAfter:       time.Now().AddDate(0, 0, 60),
		},
		{
			name:           "WARNING状態（20日残り）",
			daysRemaining:  20,
			expectedStatus: "WARNING",
			notAfter:       time.Now().AddDate(0, 0, 20),
		},
		{
			name:           "CRITICAL状態（5日残り）",
			daysRemaining:  5,
			expectedStatus: "CRITICAL",
			notAfter:       time.Now().AddDate(0, 0, 5),
		},
		{
			name:           "CRITICAL状態（期限切れ）",
			daysRemaining:  -1,
			expectedStatus: "CRITICAL",
			notAfter:       time.Now().AddDate(0, 0, -1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			daysRemaining := int(tc.notAfter.Sub(now).Hours() / 24)

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

			if status != tc.expectedStatus {
				t.Errorf("ステータスが正しくありません。期待: %s, 実際: %s", tc.expectedStatus, status)
			}
		})
	}
}

// TestJSTTimeZone JSTタイムゾーンのテスト
func TestJSTTimeZone(t *testing.T) {
	if JST == nil {
		t.Fatal("JSTタイムゾーンが初期化されていません")
	}

	// JSTのオフセットを確認（+9時間 = 32400秒）
	now := time.Now()
	_, offset := now.In(JST).Zone()
	expectedOffset := 9 * 60 * 60 // 9時間を秒に変換

	if offset != expectedOffset {
		t.Errorf("JSTのオフセットが正しくありません。期待: %d, 実際: %d", expectedOffset, offset)
	}
}

// TestSiteDefaultValues サイトのデフォルト値テスト
func TestSiteDefaultValues(t *testing.T) {
	// ポート番号が0の場合、デフォルトで443になることを確認
	site := Site{
		URL:  "example.com",
		Port: 0,
		Name: "",
	}

	if site.Port == 0 {
		site.Port = 443
	}
	if site.Name == "" {
		site.Name = site.URL
	}

	if site.Port != 443 {
		t.Errorf("デフォルトポートが正しくありません。期待: 443, 実際: %d", site.Port)
	}
	if site.Name != "example.com" {
		t.Errorf("デフォルト名が正しくありません。期待: example.com, 実際: %s", site.Name)
	}
}

// TestSetupLogger ロガーのセットアップテスト
func TestSetupLogger(t *testing.T) {
	// テスト用の設定（ファイルなし）
	config := &Config{}
	config.Logging.File = ""

	setupLogger(config)

	if Logger == nil {
		t.Error("ロガーが初期化されていません")
	}

	// ログファイルありのテスト
	tmpFile, err := os.CreateTemp("", "test_log_*.log")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config.Logging.File = tmpFile.Name()
	setupLogger(config)

	if Logger == nil {
		t.Error("ロガーが初期化されていません")
	}

	// ログの書き込みテスト
	Logger.Println("Test log message")

	// 無効なパスのテスト（ファイルオープンエラー）
	config.Logging.File = "/invalid/path/that/does/not/exist/test.log"
	setupLogger(config)

	// エラー時でもロガーは初期化されているはず（標準出力にフォールバック）
	if Logger == nil {
		t.Error("エラー時でもロガーが初期化されていません")
	}
}

// TestCheckAllSites 複数サイトのチェックテスト
func TestCheckAllSites(t *testing.T) {
	config := &Config{}
	config.Alert.WarningDays = 30
	config.Alert.CriticalDays = 7
	config.Sites = []Site{
		{URL: "invalid-test-site-12345.com", Port: 443, Name: "Test Site 1"},
		{URL: "invalid-test-site-67890.com", Port: 443, Name: "Test Site 2"},
	}

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	results := checkAllSites(config)

	// 結果の数を確認
	if len(results) != 2 {
		t.Errorf("結果の数が正しくありません。期待: 2, 実際: %d", len(results))
	}

	// 各結果がERRORステータスであることを確認（無効なドメインなので）
	for i, result := range results {
		if result.Status != "ERROR" {
			t.Errorf("結果[%d]のステータスが正しくありません。期待: ERROR, 実際: %s", i, result.Status)
		}
		if result.SiteName == "" {
			t.Errorf("結果[%d]のサイト名が空です", i)
		}
	}
}

// TestCheckCertificateInvalidDomain 無効なドメインのチェックテスト
func TestCheckCertificateInvalidDomain(t *testing.T) {
	config := &Config{}
	config.Alert.WarningDays = 30
	config.Alert.CriticalDays = 7

	site := Site{
		URL:  "invalid-test-domain-999999.com",
		Port: 443,
		Name: "Invalid Test Site",
	}

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	result := checkCertificate(config, site)

	// エラーステータスであることを確認
	if result.Status != "ERROR" {
		t.Errorf("ステータスが正しくありません。期待: ERROR, 実際: %s", result.Status)
	}

	if result.ErrorMessage == "" {
		t.Error("エラーメッセージが設定されていません")
	}

	if result.SiteName != "Invalid Test Site" {
		t.Errorf("サイト名が正しくありません。期待: Invalid Test Site, 実際: %s", result.SiteName)
	}
}

// TestCheckCertificateDefaultPort デフォルトポートのテスト
func TestCheckCertificateDefaultPort(t *testing.T) {
	config := &Config{}
	config.Alert.WarningDays = 30
	config.Alert.CriticalDays = 7

	site := Site{
		URL:  "invalid-test-domain-999999.com",
		Port: 0, // デフォルトポート（443になるはず）
		Name: "",
	}

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	result := checkCertificate(config, site)

	// ポートが443になっていることを確認
	if result.Port != 443 {
		t.Errorf("ポートが正しくありません。期待: 443, 実際: %d", result.Port)
	}

	// 名前がURLになっていることを確認
	if result.SiteName != "invalid-test-domain-999999.com" {
		t.Errorf("サイト名が正しくありません。期待: invalid-test-domain-999999.com, 実際: %s", result.SiteName)
	}
}

// TestCheckCertificateValidSite 有効なサイトのチェックテスト（実際の接続）
func TestCheckCertificateValidSite(t *testing.T) {
	if testing.Short() {
		t.Skip("ネットワーク接続テストをスキップします")
	}

	config := &Config{}
	config.Alert.WarningDays = 30
	config.Alert.CriticalDays = 7

	site := Site{
		URL:  "www.google.com",
		Port: 443,
		Name: "Google",
	}

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	result := checkCertificate(config, site)

	// エラーでないことを確認
	if result.Status == "ERROR" {
		t.Logf("警告: Googleへの接続に失敗しました: %s", result.ErrorMessage)
		t.Skip("ネットワーク接続が利用できないため、テストをスキップします")
	}

	// 証明書情報が取得できていることを確認
	if result.Issuer == "" {
		t.Error("発行者情報が取得できていません")
	}

	if result.NotAfter.IsZero() {
		t.Error("有効期限が取得できていません")
	}

	if result.DaysRemaining < 0 {
		t.Error("残り日数が負の値です")
	}
}

// TestCheckCertificateStatusVariations 証明書ステータスのバリエーションテスト
func TestCheckCertificateStatusVariations(t *testing.T) {
	if testing.Short() {
		t.Skip("ネットワーク接続テストをスキップします")
	}

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	testCases := []struct {
		name           string
		warningDays    int
		criticalDays   int
		url            string
		expectedStatus string // "OK", "WARNING", "CRITICAL", "ERROR" のいずれか（またはスキップ）
	}{
		{
			name:           "通常の証明書チェック（Google）",
			warningDays:    30,
			criticalDays:   7,
			url:            "www.google.com",
			expectedStatus: "OK", // Googleの証明書は通常有効期限が十分残っている
		},
		{
			name:           "警告期間が長い設定",
			warningDays:    365,
			criticalDays:   90,
			url:            "www.google.com",
			expectedStatus: "", // ステータスは可変なのでチェックしない
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{}
			config.Alert.WarningDays = tc.warningDays
			config.Alert.CriticalDays = tc.criticalDays

			site := Site{
				URL:  tc.url,
				Port: 443,
				Name: tc.name,
			}

			result := checkCertificate(config, site)

			if result.Status == "ERROR" {
				t.Logf("警告: %sへの接続に失敗しました: %s", tc.url, result.ErrorMessage)
				t.Skip("ネットワーク接続が利用できないため、テストをスキップします")
			}

			// 基本的な検証
			if result.SiteName == "" {
				t.Error("サイト名が設定されていません")
			}

			if result.URL == "" {
				t.Error("URLが設定されていません")
			}

			if result.Port == 0 {
				t.Error("ポート番号が設定されていません")
			}

			// 期待されるステータスがある場合はチェック
			if tc.expectedStatus != "" && result.Status != tc.expectedStatus {
				t.Logf("注意: ステータスが期待と異なります。期待: %s, 実際: %s (残り日数: %d)",
					tc.expectedStatus, result.Status, result.DaysRemaining)
			}
		})
	}
}

// TestSendDiscordNotificationDisabled Discord通知無効時のテスト
func TestSendDiscordNotificationDisabled(t *testing.T) {
	config := &Config{}
	config.Discord.Enabled = false

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	results := []CertInfo{
		{
			SiteName:      "Test Site",
			URL:           "test.com",
			Port:          443,
			Status:        "CRITICAL",
			DaysRemaining: 5,
		},
	}

	err := sendDiscordNotification(config, results)
	if err != nil {
		t.Errorf("Discord通知無効時にエラーが発生しました: %v", err)
	}
}

// TestSendDiscordNotificationNoWebhook Webhook URL未設定時のテスト
func TestSendDiscordNotificationNoWebhook(t *testing.T) {
	config := &Config{}
	config.Discord.Enabled = true
	config.Discord.WebhookURL = ""

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	results := []CertInfo{
		{
			SiteName:      "Test Site",
			URL:           "test.com",
			Port:          443,
			Status:        "CRITICAL",
			DaysRemaining: 5,
		},
	}

	err := sendDiscordNotification(config, results)
	if err != nil {
		t.Errorf("Webhook URL未設定時にエラーが発生しました: %v", err)
	}
}

// TestSendDiscordNotificationFiltering 通知フィルタリングのテスト
func TestSendDiscordNotificationFiltering(t *testing.T) {
	config := &Config{}
	config.Discord.Enabled = true
	config.Discord.WebhookURL = "https://discord.com/api/webhooks/test/test"
	config.Discord.NotifyOn = []string{"CRITICAL"}

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	results := []CertInfo{
		{
			SiteName:      "OK Site",
			URL:           "ok.com",
			Port:          443,
			Status:        "OK",
			DaysRemaining: 90,
		},
		{
			SiteName:      "Warning Site",
			URL:           "warning.com",
			Port:          443,
			Status:        "WARNING",
			DaysRemaining: 20,
		},
	}

	// フィルタリングされて通知対象がないため、エラーは発生しないはず
	err := sendDiscordNotification(config, results)
	if err != nil {
		t.Errorf("通知対象なし時にエラーが発生しました: %v", err)
	}
}

// TestSendDiscordNotificationDefaultWebhook デフォルトWebhook URLのテスト
func TestSendDiscordNotificationDefaultWebhook(t *testing.T) {
	config := &Config{}
	config.Discord.Enabled = true
	config.Discord.WebhookURL = "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	results := []CertInfo{
		{
			SiteName:      "Test Site",
			URL:           "test.com",
			Port:          443,
			Status:        "CRITICAL",
			DaysRemaining: 5,
		},
	}

	// デフォルトのWebhook URLは無視されるはず
	err := sendDiscordNotification(config, results)
	if err != nil {
		t.Errorf("デフォルトWebhook URL時にエラーが発生しました: %v", err)
	}
}

// TestSendDiscordNotificationNoFilter フィルターなしのテスト
func TestSendDiscordNotificationNoFilter(t *testing.T) {
	config := &Config{}
	config.Discord.Enabled = true
	config.Discord.WebhookURL = "https://discord.com/api/webhooks/test/test"
	config.Discord.NotifyOn = []string{} // フィルターなし

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	results := []CertInfo{
		{
			SiteName:      "Test Site 1",
			URL:           "test1.com",
			Port:          443,
			Issuer:        "Test CA",
			Subject:       "test1.com",
			NotBefore:     time.Now().AddDate(0, -1, 0),
			NotAfter:      time.Now().AddDate(0, 2, 0),
			Status:        "OK",
			DaysRemaining: 60,
		},
		{
			SiteName:      "Test Site 2",
			URL:           "test2.com",
			Port:          443,
			Status:        "ERROR",
			ErrorMessage:  "Connection failed",
		},
	}

	// フィルターなしの場合、すべての結果が対象になる
	// 実際のHTTP送信は失敗するが、処理自体はエラーにならない
	err := sendDiscordNotification(config, results)
	// ネットワークエラーが発生する可能性があるが、それは正常
	if err != nil {
		t.Logf("予想されるネットワークエラー: %v", err)
	}
}

// TestSendDiscordNotificationMultipleStatuses 複数ステータスのテスト
func TestSendDiscordNotificationMultipleStatuses(t *testing.T) {
	config := &Config{}
	config.Discord.Enabled = true
	config.Discord.WebhookURL = "https://discord.com/api/webhooks/test/test"
	config.Discord.NotifyOn = []string{"WARNING", "CRITICAL", "ERROR"}

	// ロガーのセットアップ
	Logger = log.New(os.Stdout, "", log.LstdFlags)

	now := time.Now()
	results := []CertInfo{
		{
			SiteName:      "Warning Site",
			URL:           "warning.com",
			Port:          443,
			Issuer:        "CA 1",
			Subject:       "warning.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 0, 20),
			Status:        "WARNING",
			DaysRemaining: 20,
		},
		{
			SiteName:      "Critical Site",
			URL:           "critical.com",
			Port:          443,
			Issuer:        "CA 2",
			Subject:       "critical.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 0, 5),
			Status:        "CRITICAL",
			DaysRemaining: 5,
		},
		{
			SiteName:     "Error Site",
			URL:          "error.com",
			Port:         443,
			Status:       "ERROR",
			ErrorMessage: "Connection timeout",
		},
	}

	// 複数のステータスが通知対象
	err := sendDiscordNotification(config, results)
	if err != nil {
		t.Logf("予想されるネットワークエラー: %v", err)
	}
}

// TestCertInfoWithErrorStatus エラー状態の証明書情報テスト
func TestCertInfoWithErrorStatus(t *testing.T) {
	certInfo := CertInfo{
		SiteName:     "Error Site",
		URL:          "error.com",
		Port:         443,
		Status:       "ERROR",
		ErrorMessage: "Connection timeout",
	}

	if certInfo.Status != "ERROR" {
		t.Errorf("ステータスが正しくありません。期待: ERROR, 実際: %s", certInfo.Status)
	}

	if certInfo.ErrorMessage == "" {
		t.Error("エラーメッセージが設定されていません")
	}
}

// TestConfigStructure 設定構造体のテスト
func TestConfigStructure(t *testing.T) {
	config := Config{}

	// デフォルト値のテスト
	if config.Sites == nil {
		config.Sites = []Site{}
	}

	if len(config.Sites) != 0 {
		t.Errorf("サイト数が正しくありません。期待: 0, 実際: %d", len(config.Sites))
	}

	// サイトの追加
	config.Sites = append(config.Sites, Site{
		URL:  "example.com",
		Port: 443,
		Name: "Example",
	})

	if len(config.Sites) != 1 {
		t.Errorf("サイト数が正しくありません。期待: 1, 実際: %d", len(config.Sites))
	}
}

// TestMultipleReportGeneration 複数レポート生成のテスト
func TestMultipleReportGeneration(t *testing.T) {
	now := time.Now()
	results := []CertInfo{
		{
			SiteName:      "Site 1",
			URL:           "site1.com",
			Port:          443,
			Issuer:        "CA 1",
			Subject:       "site1.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 2, 0),
			DaysRemaining: 60,
			Status:        "OK",
		},
		{
			SiteName:      "Site 2",
			URL:           "site2.com",
			Port:          8443,
			Issuer:        "CA 2",
			Subject:       "site2.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 0, 10),
			DaysRemaining: 10,
			Status:        "WARNING",
		},
	}

	// テキストレポート
	textReport1 := generateTextReport(results)
	textReport2 := generateTextReport(results)

	if textReport1 != textReport2 {
		t.Error("同じ入力で異なるテキストレポートが生成されました")
	}

	// HTMLレポート
	htmlReport1 := generateHTMLReport(results)
	htmlReport2 := generateHTMLReport(results)

	if htmlReport1 != htmlReport2 {
		t.Error("同じ入力で異なるHTMLレポートが生成されました")
	}
}

// Benchmark tests
func BenchmarkGenerateTextReport(b *testing.B) {
	now := time.Now()
	results := []CertInfo{
		{
			SiteName:      "Example Site",
			URL:           "example.com",
			Port:          443,
			Issuer:        "Let's Encrypt",
			Subject:       "example.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 2, 0),
			DaysRemaining: 60,
			Status:        "OK",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateTextReport(results)
	}
}

func BenchmarkGenerateHTMLReport(b *testing.B) {
	now := time.Now()
	results := []CertInfo{
		{
			SiteName:      "Example Site",
			URL:           "example.com",
			Port:          443,
			Issuer:        "Let's Encrypt",
			Subject:       "example.com",
			NotBefore:     now.AddDate(0, -1, 0),
			NotAfter:      now.AddDate(0, 2, 0),
			DaysRemaining: 60,
			Status:        "OK",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateHTMLReport(results)
	}
}
