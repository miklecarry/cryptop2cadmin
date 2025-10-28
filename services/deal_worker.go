package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"hostmanager/models"

	"github.com/beego/beego/v2/client/orm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/skip2/go-qrcode"
)

var (
	// Кэш отправленных сделок: dealID -> время отправки
	sentDeals     = make(map[int64]time.Time)
	sentDealsMu   sync.Mutex
	cleanupTicker *time.Ticker
)

// StartSimpleMonitoring запускает простой мониторинг без воркеров
func StartSimpleMonitoring() {
	// Запускаем очистку старых записей каждую минуту
	cleanupTicker = time.NewTicker(1 * time.Minute)
	go cleanupOldDeals()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		checkAllHosts()
	}
}

// cleanupOldDeals очищает старые записи из кэша (старше 2 минут)
func cleanupOldDeals() {
	for range cleanupTicker.C {
		sentDealsMu.Lock()
		now := time.Now()
		for dealID, sentTime := range sentDeals {
			if now.Sub(sentTime) > 2*time.Minute {
				delete(sentDeals, dealID)
			}
		}
		sentDealsMu.Unlock()
		log.Printf("Очистка кэша сделок. Осталось: %d", len(sentDeals))
	}
}

// isDealSent проверяет, отправлялась ли уже сделка
func isDealSent(dealID int64) bool {
	sentDealsMu.Lock()
	defer sentDealsMu.Unlock()

	_, exists := sentDeals[dealID]
	return exists
}

// markDealSent помечает сделку как отправленную
func markDealSent(dealID int64) {
	sentDealsMu.Lock()
	defer sentDealsMu.Unlock()

	sentDeals[dealID] = time.Now()
}

// removeDealSent удаляет сделку из кэша (можно вызывать при завершении сделки)
func removeDealSent(dealID int64) {
	sentDealsMu.Lock()
	defer sentDealsMu.Unlock()

	delete(sentDeals, dealID)
}

func checkAllHosts() {
	o := orm.NewOrm()
	var hosts []models.Host

	// Получаем все активные хосты с пользователями
	_, err := o.QueryTable("host").
		Filter("Active", true).
		RelatedSel("User").
		All(&hosts)

	if err != nil {
		log.Printf("Ошибка получения хостов: %v", err)
		return
	}

	// Для каждого активного хоста проверяем сделки
	for _, host := range hosts {
		checkDealsForHost(host)
	}
}

func checkDealsForHost(host models.Host) {
	// Проверяем, что есть chatID
	if host.User == nil || host.User.TelegramChatID == 0 {
		return
	}

	// Делаем запрос к API
	url := "https://app.cr.bot/internal/v1/p2c/payments?size=40&status=processing"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", "access_token="+host.AccessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Хост %d: http error: %v", host.Id, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Хост %d: status %d", host.Id, resp.StatusCode)
		return
	}

	var result struct {
		Data []DealPreview `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Хост %d: decode error: %v", host.Id, err)
		return
	}

	// Обрабатываем сделки
	for _, deal := range result.Data {
		// Проверяем, не отправляли ли уже эту сделку
		if isDealSent(deal.ID) {
			continue
		}

		processDeal(host, deal)
	}
}

func processDeal(host models.Host, deal DealPreview) {
	details, err := getDealDetails(deal.ID, host.AccessToken)
	if err != nil {
		log.Printf("Хост %d: getDealDetails error for %d: %v", host.Id, deal.ID, err)
		return
	}

	// Отправляем сообщение в Telegram
	sendTelegramDealMessage(host.User.TelegramChatID, details, host.Id)

	// Помечаем сделку как отправленную
	markDealSent(deal.ID)

	// Логируем
	o := orm.NewOrm()
	hostLog := models.HostLog{
		Host:  &host,
		Level: "bounty",
		Message: fmt.Sprintf("https://app.cr.bot/p2c/orders/%d  Сумма: %s %s Магазин: %s",
			details.ID, details.InAmount, details.InAsset, details.BrandName),
	}
	if _, err := o.Insert(&hostLog); err != nil {
		log.Printf("Хост %d: HostLog insert error: %v", host.Id, err)
	}
}

// Остальные функции без изменений...
func getDealDetails(dealID int64, accessToken string) (*DealDetails, error) {
	url := fmt.Sprintf("https://app.cr.bot/internal/v1/p2c/payments/%d", dealID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", "access_token="+accessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var result struct {
		Data DealDetails `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func sendTelegramDealMessage(chatID int64, deal *DealDetails, hostID int64) {
	if Bot == nil {
		return
	}

	text := fmt.Sprintf("https://app.cr.bot/p2c/orders/%d\nСумма: %s %s\nМагазин: %s",
		deal.ID, deal.InAmount, deal.InAsset, deal.BrandName)

	// Генерация QR-кода
	var photoFile tgbotapi.FileBytes
	png, err := qrcode.Encode(deal.Url, qrcode.Medium, 256)
	if err == nil {
		photoFile = tgbotapi.FileBytes{
			Name:  fmt.Sprintf("qr%d.png", deal.ID),
			Bytes: png,
		}
	}

	callbackData := fmt.Sprintf("complete_%d_%d", hostID, deal.ID)
	btn := tgbotapi.NewInlineKeyboardButtonData("Оплатил", callbackData)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btn),
	)

	// Отправка с фото если QR сгенерировался
	if len(photoFile.Bytes) > 0 {
		photo := tgbotapi.NewPhoto(chatID, photoFile)
		_, err := Bot.Send(photo)

		if err != nil {
			log.Printf("sendTelegramMessage: send photo failed: %v", err)
		} else {
			// После фото отправляем текст с кнопкой
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ReplyMarkup = keyboard
			Bot.Send(msg)
			return
		}
	}

	// Если фото не получилось — отправляем просто текст с кнопкой
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	Bot.Send(msg)
}

// Структуры
type DealPreview struct {
	ID     int64  `json:"id"`
	Asset  string `json:"asset"`
	Fiat   string `json:"fiat"`
	Amount string `json:"amount"`
	Status string `json:"status"`
}

type DealDetails struct {
	ID           int64  `json:"id"`
	BrandName    string `json:"brand_name"`
	InAmount     string `json:"in_amount"`
	InAsset      string `json:"in_asset"`
	OutAmount    string `json:"out_amount"`
	OutAsset     string `json:"out_asset"`
	ExchangeRate string `json:"exchange_rate"`
	RewardAmount string `json:"reward_amount"`
	Status       string `json:"status"`
	Url          string `json:"url"`
}
