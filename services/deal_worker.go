package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"hostmanager/models"

	"github.com/beego/beego/v2/client/orm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	qrcode "github.com/skip2/go-qrcode"
)

var (
	workers   = make(map[int64]*DealWorker)
	workersMu sync.Mutex
)

type DealWorker struct {
	Host        models.Host
	LastCursor  string
	ActiveDeals map[int64]string // dealID → messageID
	cancel      context.CancelFunc
}

func StartDealWorker(host models.Host) {
	workersMu.Lock()
	defer workersMu.Unlock()

	if _, exists := workers[host.Id]; exists {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &DealWorker{
		Host:        host,
		ActiveDeals: make(map[int64]string),
		cancel:      cancel,
	}
	workers[host.Id] = w

	// Очистка сообщений
	if host.User != nil && host.User.TelegramChatID != 0 {
		ClearUserMessages(host.User.TelegramChatID, w)
	}

	go w.run(ctx)
}

func (w *DealWorker) run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Воркер для хоста %d остановлен", w.Host.Id)
			return
		case <-ticker.C:
			w.checkDeals()
		}
	}
}
func (w *DealWorker) stop() {
	w.cancel()
}
func StopDealWorker(hostID int64) {
	workersMu.Lock()
	defer workersMu.Unlock()

	if w, exists := workers[hostID]; exists {
		w.stop()
		delete(workers, hostID)
		log.Printf("Воркер для хоста %d остановлен", hostID)
	}
}

func (w *DealWorker) checkDeals() {
	url := "https://app.cr.bot/internal/v1/p2c/payments?size=40&status=processing"
	if w.LastCursor != "" {
		url += "&cursor=" + w.LastCursor
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", "access_token="+w.Host.AccessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var result struct {
		Data   []DealPreview `json:"data"`
		Cursor string        `json:"cursor"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	w.LastCursor = result.Cursor

	o := orm.NewOrm()
	o.LoadRelated(&w.Host, "User")
	if w.Host.User == nil || w.Host.User.TelegramChatID == 0 {
		return
	}
	chatID := w.Host.User.TelegramChatID

	currentIDs := make(map[int64]struct{})
	for _, d := range result.Data {
		currentIDs[d.ID] = struct{}{}
	}

	// 1️⃣ Удаляем оплаченные
	for id, msgID := range w.ActiveDeals {
		if _, stillActive := currentIDs[id]; !stillActive {
			// сделка пропала — удалить из Telegram
			msg := tgbotapi.NewDeleteMessage(chatID, toInt(msgID))
			Bot.Send(msg)
			delete(w.ActiveDeals, id)
		}
	}

	// 2️⃣ Добавляем новые
	for _, deal := range result.Data {
		if _, exists := w.ActiveDeals[deal.ID]; exists {
			continue
		}
		details, err := w.getDealDetails(deal.ID)
		if err != nil {
			continue
		}
		msgID := w.sendTelegramMessage(chatID, details)
		if msgID != 0 {
			w.ActiveDeals[deal.ID] = fmt.Sprintf("%d", msgID)
		}

		hostLog := models.HostLog{
			Host:  &w.Host,
			Level: "bounty",
			Message: fmt.Sprintf("https://app.cr.bot/p2c/orders/%d  Сумма: %s %s Магазин: %s",
				details.ID, details.InAmount, details.InAsset, details.BrandName),
		}
		o.Insert(&hostLog)
	}
}

func toInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func ClearUserMessages(chatID int64, w *DealWorker) {
	for _, msgID := range w.ActiveDeals {
		msg := tgbotapi.NewDeleteMessage(chatID, toInt(msgID))
		Bot.Send(msg)
	}
	w.ActiveDeals = make(map[int64]string)
}
func (w *DealWorker) getDealDetails(dealID int64) (*DealDetails, error) {
	url := fmt.Sprintf("https://app.cr.bot/internal/v1/p2c/payments/%d", dealID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", "access_token="+w.Host.AccessToken)

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
	json.NewDecoder(resp.Body).Decode(&result)
	return &result.Data, nil
}

func (w *DealWorker) sendTelegramMessage(chatID int64, deal *DealDetails) int {
	if Bot == nil {
		return 0
	}

	// 1️⃣ Генерация QR-кода
	png, err := qrcode.Encode(deal.Url, qrcode.Medium, 256)
	if err == nil {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
			Name:  fmt.Sprintf("qr_%d.png", deal.ID),
			Bytes: png,
		})
		Bot.Send(photo)
	}

	// 2️⃣ Основной текст сделки
	text := fmt.Sprintf(
		"💳 <b>Новая сделка</b>\n"+
			"https://app.cr.bot/p2c/orders/%d\n"+
			"Сумма: %s %s\nМагазин: %s\n\n"+
			"🔗 <a href=\"%s\">Ссылка на оплату</a>",
		deal.ID, deal.InAmount, deal.InAsset, deal.BrandName, deal.Url)

	callbackData := fmt.Sprintf("complete_%d_%d", w.Host.Id, deal.ID)
	btn := tgbotapi.NewInlineKeyboardButtonData("✅ Оплатил", callbackData)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btn),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	sent, err := Bot.Send(msg)
	if err != nil {
		return 0
	}
	return sent.MessageID
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
