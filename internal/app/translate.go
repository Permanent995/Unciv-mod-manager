package app

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	translateMicrosoft = "microsoft"
	translateYandex    = "yandex"
	translateCustom    = "custom"
)

var translateClient = &http.Client{Timeout: 15 * time.Second}

// ── Public API ────────────────────────────────────────────────────────

// TranslateText translates text to Chinese using the configured provider.
func (a *App) TranslateText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("文本为空")
	}
	provider := a.config.TranslateProvider
	if provider == "" {
		provider = translateMicrosoft
	}
	switch provider {
	case translateYandex:
		return translateWithYandex(text)
	case translateCustom:
		return a.translateWithCustom(text)
	default:
		return translateWithMicrosoft(text)
	}
}

// ── Microsoft Translator (free, no API key) ──────────────────────────

var microsoftBaseURL = "https://api.cognitive.microsofttranslator.com"

func translateWithMicrosoft(text string) (string, error) {
	endpoint := strings.TrimRight(microsoftBaseURL, "/") + "/translate?api-version=3.0&to=" + url.QueryEscape("zh-Hans")
	sigPath, _ := signPathFromURL(endpoint)

	payload, _ := json.Marshal([]map[string]string{{"Text": text}})
	req, _ := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("X-MT-Signature", getMSSignature(sigPath))

	resp, err := translateClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", translateErr(resp)
	}
	var result []struct {
		Translations []struct{ Text string `json:"text"` } `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result) == 0 || len(result[0].Translations) == 0 {
		return "", fmt.Errorf("翻译未返回结果")
	}
	return result[0].Translations[0].Text, nil
}

// ── Yandex Translator (free, no API key) ─────────────────────────────

var yandexBaseURL = "https://translate.yandex.net/api/v1/tr.json"

func translateWithYandex(text string) (string, error) {
	endpoint := strings.TrimRight(yandexBaseURL, "/") + "/translate"
	params := url.Values{}
	params.Set("ucid", genUCID())
	params.Set("srv", "android")
	params.Set("format", "text")
	endpoint += "?" + params.Encode()

	form := url.Values{}
	form.Set("text", text)
	form.Set("lang", "en-zh")

	req, _ := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "ru.yandex.translate/3.20.2024")

	resp, err := translateClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", translateErr(resp)
	}
	var result struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Text    []string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Code != 0 && result.Code != http.StatusOK {
		return "", fmt.Errorf("Yandex 翻译失败: %s", result.Message)
	}
	if len(result.Text) == 0 {
		return "", fmt.Errorf("翻译未返回结果")
	}
	return result.Text[0], nil
}

// ── Custom AI (OpenAI-compatible) ─────────────────────────────────────

func (a *App) translateWithCustom(text string) (string, error) {
	baseURL := a.config.TranslateCustomURL
	apiKey := a.config.TranslateCustomKey
	model := a.config.TranslateCustomModel
	if baseURL == "" {
		return "", fmt.Errorf("未设置翻译 API 地址")
	}
	if apiKey == "" {
		return "", fmt.Errorf("未设置翻译 API Key")
	}
	if model == "" {
		model = "deepseek-chat"
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	payload, _ := json.Marshal(map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一位专业翻译。将以下内容翻译成中文，保持语义准确、语言自然。直接返回翻译结果，不要添加解释。"},
			{"role": "user", "content": text},
		},
		"temperature": 0.3,
	})

	req, _ := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", translateErr(resp)
	}
	var result struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("翻译未返回结果")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// ── Helpers ───────────────────────────────────────────────────────────

func signPathFromURL(raw string) (string, error) {
	p, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return p.Host + p.RequestURI(), nil
}

func getMSSignature(path string) string {
	guid := genUCID()
	escaped := url.QueryEscape(path)
	ts := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05GMT")
	text := strings.ToLower("MSTranslatorAndroidApp" + escaped + ts + guid)

	mac := hmac.New(sha256.New, msKey[:])
	mac.Write([]byte(text))
	hash := mac.Sum(nil)
	return "MSTranslatorAndroidApp::" + base64.StdEncoding.EncodeToString(hash) + "::" + ts + "::" + guid
}

func genUCID() string {
	var b [16]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func translateErr(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return fmt.Errorf("翻译接口返回 %d", resp.StatusCode)
	}
	return fmt.Errorf("翻译接口返回 %d: %s", resp.StatusCode, msg)
}

// Microsoft Translator Android app signing key (publicly known, embedded in the app).
var msKey = [64]byte{
	0xa2, 0x29, 0x3a, 0x3d, 0xd0, 0xdd, 0x32, 0x73,
	0x97, 0x7a, 0x64, 0xdb, 0xc2, 0xf3, 0x27, 0xf5,
	0xd7, 0xbf, 0x87, 0xd9, 0x45, 0x9d, 0xf0, 0x5a,
	0x09, 0x66, 0xc6, 0x30, 0xc6, 0x6a, 0xaa, 0x84,
	0x9a, 0x41, 0xaa, 0x94, 0x3a, 0xa8, 0xd5, 0x1a,
	0x6e, 0x4d, 0xaa, 0xc9, 0xa3, 0x70, 0x12, 0x35,
	0xc7, 0xeb, 0x12, 0xf6, 0xe8, 0x23, 0x07, 0x9e,
	0x47, 0x10, 0x95, 0x91, 0x88, 0x55, 0xd8, 0x17,
}
