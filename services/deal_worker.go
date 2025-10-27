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
	workersMu sync.RWMutex
)

type DealWorker struct {
	Host           models.Host
	LastCursor     string
	ActiveDeals    map[int64]string // dealID → messageID
	ProcessedDeals sync.Map         // dealID → timestamp (для отслеживания уже обработанных)
	cancel         context.CancelFunc
	processingMu   sync.Mutex // Блокировка для обработки сделок
}

func StartDealWorker(host models.Host) {
	workersMu.Lock()
	defer workersMu.Unlock()

	// Останавливаем старый воркер если существует
	if oldWorker, exists := workers[host.Id]; exists {
		oldWorker.stop()
		delete(workers, host.Id)
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
	log.Printf("Воркер для хоста %d запущен", host.Id)
}

func (w *DealWorker) run(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second) // Увеличил интервал для снижения нагрузки
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
	if w.cancel != nil {
		w.cancel()
	}
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
	w.processingMu.Lock()
	defer w.processingMu.Unlock()

	url := "https://app.cr.bot/internal/v1/p2c/payments?size=20&status=processing" // Уменьшил размер
	if w.LastCursor != "" {
		url += "&cursor=" + w.LastCursor
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Ошибка создания запроса: %v", err)
		return
	}
	req.Header.Set("Cookie", "access_token="+w.Host.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ошибка запроса для хоста %d: %v", w.Host.Id, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Неверный статус код для хоста %d: %d", w.Host.Id, resp.StatusCode)
		return
	}

	var result struct {
		Data   []DealPreview `json:"data"`
		Cursor string        `json:"cursor"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Ошибка декодирования JSON для хоста %d: %v", w.Host.Id, err)
		return
	}

	// Обновляем курсор только при успешном получении данных
	if result.Cursor != "" {
		w.LastCursor = result.Cursor
	}

	// Загружаем данные пользователя
	o := orm.NewOrm()
	if err := o.Read(&w.Host); err != nil {
		log.Printf("Ошибка чтения хоста %d: %v", w.Host.Id, err)
		return
	}

	if w.Host.User == nil {
		if err := o.LoadRelated(&w.Host, "User"); err != nil || w.Host.User == nil {
			log.Printf("Пользователь не найден для хоста %d", w.Host.Id)
			return
		}
	}

	if w.Host.User.TelegramChatID == 0 {
		log.Printf("TelegramChatID не установлен для пользователя хоста %d", w.Host.Id)
		return
	}

	chatID := w.Host.User.TelegramChatID

	// Собираем текущие активные сделки
	currentDealIDs := make(map[int64]bool)
	for _, deal := range result.Data {
		currentDealIDs[deal.ID] = true
	}

	// Удаляем завершенные сделки
	for dealID, msgID := range w.ActiveDeals {
		if !currentDealIDs[dealID] {
			// Сделка завершена - удаляем сообщение
			if err := w.deleteTelegramMessage(chatID, msgID); err == nil {
				delete(w.ActiveDeals, dealID)
				w.ProcessedDeals.Delete(dealID)
				log.Printf("Удалена завершенная сделка %d для хоста %d", dealID, w.Host.Id)
			}
		}
	}

	// Обрабатываем новые сделки
	for _, deal := range result.Data {
		// Проверяем, не обрабатывали ли мы уже эту сделку
		if _, processed := w.ProcessedDeals.Load(deal.ID); processed {
			continue
		}

		// Проверяем, не активна ли уже сделка
		if _, isActive := w.ActiveDeals[deal.ID]; isActive {
			continue
		}

		// Получаем детали сделки
		details, err := w.getDealDetails(deal.ID)
		if err != nil {
			log.Printf("Ошибка получения деталей сделки %d: %v", deal.ID, err)
			continue
		}

		// Отправляем сообщение в Telegram
		msgID, err := w.sendTelegramMessage(chatID, details)
		if err != nil {
			log.Printf("Ошибка отправки сообщения для сделки %d: %v", deal.ID, err)
			continue
		}

		// Сохраняем информацию о сделке
		w.ActiveDeals[deal.ID] = fmt.Sprintf("%d", msgID)
		w.ProcessedDeals.Store(deal.ID, time.Now())

		// Логируем в базу
		hostLog := models.HostLog{
			Host:  &models.Host{Id: w.Host.Id},
			Level: "bounty",
			Message: fmt.Sprintf("https://app.cr.bot/p2c/orders/%d Сумма: %s %s Магазин: %s",
				details.ID, details.InAmount, details.InAsset, details.BrandName),
		}
		if _, err := o.Insert(&hostLog); err != nil {
			log.Printf("Ошибка логирования сделки %d: %v", deal.ID, err)
		}

		log.Printf("Обработана новая сделка %d для хоста %d", deal.ID, w.Host.Id)

		// Небольшая задержка между обработкой сделок
		time.Sleep(100 * time.Millisecond)
	}
}

func (w *DealWorker) deleteTelegramMessage(chatID int64, msgID string) error {
	if Bot == nil {
		return fmt.Errorf("бот не инициализирован")
	}

	msg := tgbotapi.NewDeleteMessage(chatID, toInt(msgID))
	_, err := Bot.Send(msg)
	return err
}

func toInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func ClearUserMessages(chatID int64, w *DealWorker) {
	for dealID, msgID := range w.ActiveDeals {
		w.deleteTelegramMessage(chatID, msgID)
		delete(w.ActiveDeals, dealID)
		w.ProcessedDeals.Delete(dealID)
	}
}

func (w *DealWorker) getDealDetails(dealID int64) (*DealDetails, error) {
	url := fmt.Sprintf("https://app.cr.bot/internal/v1/p2c/payments/%d", dealID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "access_token="+w.Host.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
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

func (w *DealWorker) sendTelegramMessage(chatID int64, deal *DealDetails) (int, error) {
	if Bot == nil {
		return 0, fmt.Errorf("бот не инициализирован")
	}

	// Генерация QR-кода (если нужно)
	var photoMsgID int
	png, err := qrcode.Encode(deal.Url, qrcode.Medium, 256)
	if err == nil {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
			Name:  fmt.Sprintf("qr_%d.png", deal.ID),
			Bytes: png,
		})
		sentPhoto, err := Bot.Send(photo)
		if err == nil {
			photoMsgID = sentPhoto.MessageID
		}
	}

	// Основное сообщение
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
		// Если основное сообщение не отправилось, удаляем фото (если было отправлено)
		if photoMsgID != 0 {
			w.deleteTelegramMessage(chatID, fmt.Sprintf("%d", photoMsgID))
		}
		return 0, err
	}

	return sent.MessageID, nil
}

// Получить воркер по ID хоста (для отладки)
func GetWorker(hostID int64) *DealWorker {
	workersMu.RLock()
	defer workersMu.RUnlock()
	return workers[hostID]
}

// Структуры остаются без изменений
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
