package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	ActiveDeals map[int64]int      // dealID → messageID (int) — changed from string
	seen        map[int64]struct{} // локальный набор уже отправленных сделок
	mu          sync.Mutex         // защита для полей воркера
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
		ActiveDeals: make(map[int64]int),
		seen:        make(map[int64]struct{}),
		cancel:      cancel,
	}
	workers[host.Id] = w

	// Очистка сообщений (делаем это потокобезопасно)
	if host.User != nil && host.User.TelegramChatID != 0 {
		w.ClearUserMessages(host.User.TelegramChatID)
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
	// Формируем URL; если LastCursor установлен — добавляем
	w.mu.Lock()
	url := "https://app.cr.bot/internal/v1/p2c/payments?size=40&status=processing"
	if w.LastCursor != "" {
		url += "&cursor=" + w.LastCursor
	}
	accessToken := w.Host.AccessToken
	w.mu.Unlock()

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", "access_token="+accessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("checkDeals: http error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("checkDeals: status %d", resp.StatusCode)
		return
	}

	var result struct {
		Data   []DealPreview `json:"data"`
		Cursor string        `json:"cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("checkDeals: decode error: %v", err)
		return
	}

	// Обновляем курсор потокобезопасно
	w.mu.Lock()
	// Если API возвращает курсор — обновляем. Дополнительно можно проверить, не пустой ли он.
	if result.Cursor != "" {
		w.LastCursor = result.Cursor
	}
	w.mu.Unlock()

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

	// 1️⃣ Удаляем оплаченные (те, которые в ActiveDeals больше не возвращаются)
	w.mu.Lock()
	for id, msgID := range w.ActiveDeals {
		if _, stillActive := currentIDs[id]; !stillActive {
			// сделка пропала — удалить из Telegram
			msg := tgbotapi.NewDeleteMessage(chatID, msgID)
			if _, err := Bot.Send(msg); err != nil {
				log.Printf("Clear Telegram message %d failed: %v", msgID, err)
			}
			delete(w.ActiveDeals, id)
			delete(w.seen, id) // можно забыть из seen, чтобы при новой идентичной сделке можно было заново отправить
		}
	}
	w.mu.Unlock()

	// 2️⃣ Добавляем новые
	for _, deal := range result.Data {
		w.mu.Lock()
		if _, exists := w.ActiveDeals[deal.ID]; exists {
			w.mu.Unlock()
			continue // уже отображается
		}
		if _, wasSeen := w.seen[deal.ID]; wasSeen {
			// уже однажды отправляли, пропускаем (защита от дубликатов API)
			w.mu.Unlock()
			continue
		}
		w.mu.Unlock()

		details, err := w.getDealDetails(deal.ID)
		if err != nil {
			log.Printf("getDealDetails error for %d: %v", deal.ID, err)
			continue
		}
		msgID := w.sendTelegramMessage(chatID, details)
		if msgID != 0 {
			w.mu.Lock()
			w.ActiveDeals[deal.ID] = msgID
			w.seen[deal.ID] = struct{}{}
			w.mu.Unlock()
		}

		hostLog := models.HostLog{
			Host:  &w.Host,
			Level: "bounty",
			Message: fmt.Sprintf("https://app.cr.bot/p2c/orders/%d  Сумма: %s %s Магазин: %s",
				details.ID, details.InAmount, details.InAsset, details.BrandName),
		}
		if _, err := o.Insert(&hostLog); err != nil {
			log.Printf("HostLog insert error: %v", err)
		}
	}
}

func ClearUserMessages(chatID int64, w *DealWorker) {
	// Удобно иметь метод, но пусть он берёт w.mu
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, msgID := range w.ActiveDeals {
		msg := tgbotapi.NewDeleteMessage(chatID, msgID)
		if _, err := Bot.Send(msg); err != nil {
			log.Printf("ClearUserMessages: failed to delete %d: %v", msgID, err)
		}
	}
	w.ActiveDeals = make(map[int64]int)
	w.seen = make(map[int64]struct{})
}

func (w *DealWorker) ClearUserMessages(chatID int64) {
	// метод-обёртка: потокобезопасно
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, msgID := range w.ActiveDeals {
		msg := tgbotapi.NewDeleteMessage(chatID, msgID)
		if _, err := Bot.Send(msg); err != nil {
			log.Printf("ClearUserMessages: failed to delete %d: %v", msgID, err)
		}
	}
	w.ActiveDeals = make(map[int64]int)
	w.seen = make(map[int64]struct{})
}

func (w *DealWorker) getDealDetails(dealID int64) (*DealDetails, error) {
	url := fmt.Sprintf("https://app.cr.bot/internal/v1/p2c/payments/%d", dealID)
	req, _ := http.NewRequest("GET", url, nil)

	w.mu.Lock()
	accessToken := w.Host.AccessToken
	w.mu.Unlock()
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

func (w *DealWorker) sendTelegramMessage(chatID int64, deal *DealDetails) int {
	if Bot == nil {
		return 0
	}

	// Генерация QR-кода (если не удалось — просто продолжим без фото)
	var photoFile tgbotapi.FileBytes
	png, err := qrcode.Encode(deal.Url, qrcode.Medium, 256)
	if err == nil {
		photoFile = tgbotapi.FileBytes{
			Name:  fmt.Sprintf("qr_%d.png", deal.ID),
			Bytes: png,
		}
	}

	// Текст для подписи (caption). Telegram captions ограничены — длину контролируйте сами.
	caption := fmt.Sprintf(
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

	if len(photoFile.Bytes) > 0 {
		photo := tgbotapi.NewPhoto(chatID, photoFile)
		photo.Caption = caption
		photo.ParseMode = "HTML"
		photo.ReplyMarkup = keyboard
		sent, err := Bot.Send(photo)
		if err != nil {
			log.Printf("sendTelegramMessage: send photo failed: %v", err)
			return 0
		}
		return sent.MessageID
	}

	// Если фото не получилось — отправим обычное сообщение с кнопкой
	msg := tgbotapi.NewMessage(chatID, caption)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	sent, err := Bot.Send(msg)
	if err != nil {
		log.Printf("sendTelegramMessage: send msg failed: %v", err)
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
