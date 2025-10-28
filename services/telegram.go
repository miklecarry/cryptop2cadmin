package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hostmanager/models"
	"hostmanager/utils"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AuthState struct {
	Step  string // "awaiting_login" или "awaiting_password"
	Login string
}

var (
	authStates = make(map[int64]*AuthState) // chat_id → состояние
	mu         sync.Mutex
)
var Bot *tgbotapi.BotAPI

func InitTelegramBot() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Println("TELEGRAM_BOT_TOKEN не задан — бот отключён")
		return
	}

	var err error
	Bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram бота: %v", err)
	}

	Bot.Debug = false
	log.Printf("Авторизован в Telegram как @%s", Bot.Self.UserName)
	go StartSimpleMonitoring()
	// Запуск в отдельной горутине
	go startTelegramPolling()
}

func startTelegramPolling() {
	if Bot == nil {
		return
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := Bot.GetUpdatesChan(u)

	for update := range updates {
		// Сначала обрабатываем callback'и (они приходят отдельно от Message)
		if update.CallbackQuery != nil {
			go handleCallbackQuery(update.CallbackQuery)
			continue
		}

		// Затем обычные сообщения
		if update.Message != nil {
			go handleMessage(update.Message)
		}
	}
}

func handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	if callback == nil {
		return
	}

	// Сразу ответим на callback, чтобы убрать "часики"
	answer := tgbotapi.NewCallback(callback.ID, "Принято")
	if _, err := Bot.Request(answer); err != nil {
		log.Printf("AnswerCallbackQuery error: %v", err)
		// дальше всё равно попробуем обработать
	}

	data := callback.Data
	if strings.HasPrefix(data, "complete_") {
		parts := strings.Split(data, "_")
		if len(parts) == 3 {
			hostID, _ := strconv.ParseInt(parts[1], 10, 64)
			dealID, _ := strconv.ParseInt(parts[2], 10, 64)

			go completePayment(hostID, dealID, callback.Message.Chat.ID)
		}
	}
}

func handleMessage(msg *tgbotapi.Message) {
	if msg == nil {
		return
	}
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	// Обработка команды /start
	if text == "/start" {
		mu.Lock()
		authStates[chatID] = &AuthState{Step: "awaiting_login"}
		mu.Unlock()
		sendMessage(chatID, "Введите ваш логин:")
		return
	}

	// Если пользователь отправил текст вида "complete_<host>_<deal>" вручную — тоже поддержим
	if strings.HasPrefix(text, "complete_") {
		parts := strings.Split(text, "_")
		if len(parts) == 3 {
			hostID, _ := strconv.ParseInt(parts[1], 10, 64)
			dealID, _ := strconv.ParseInt(parts[2], 10, 64)
			go completePayment(hostID, dealID, chatID)
			return
		}
	}

	// Получаем состояние
	mu.Lock()
	state, exists := authStates[chatID]
	mu.Unlock()

	if !exists {
		sendMessage(chatID, "Напишите /start для входа.")
		return
	}

	switch state.Step {
	case "awaiting_login":
		mu.Lock()
		state.Login = text
		state.Step = "awaiting_password"
		authStates[chatID] = state
		mu.Unlock()
		sendMessage(chatID, "Введите ваш пароль:")

	case "awaiting_password":
		mu.Lock()
		login := state.Login
		delete(authStates, chatID)
		mu.Unlock()

		password := text

		// Проверка учётных данных
		o := orm.NewOrm()
		var user models.User
		err := o.QueryTable("user").Filter("Username", login).One(&user)
		if err != nil || user.Password != models.HashPassword(password) {
			sendMessage(chatID, "❌ Неверный логин или пароль.")
			return
		}

		// ВСЕГДА обновляем chatID пользователя
		user.TelegramChatID = chatID
		user.WebAppToken = utils.GenerateToken()

		// Обновляем все необходимые поля
		if _, err := o.Update(&user, "TelegramChatID", "WebAppToken"); err != nil {
			log.Printf("Update user error: %v", err)
			sendMessage(chatID, "❌ Ошибка обновления данных.")
			return
		}

		// Формируем Web App URL
		webAppURL := os.Getenv("WEB_APP_URL") + "/login/telegram?token=" + user.WebAppToken

		// Отправляем кнопку Web App
		btn := tgbotapi.NewInlineKeyboardButtonURL("Открыть админку", webAppURL)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(btn),
		)

		msg := tgbotapi.NewMessage(chatID, "Авторизация успешна!")
		msg.ReplyMarkup = keyboard
		if sentMsg, err := Bot.Send(msg); err == nil {
			// Пинним сообщение
			pin := tgbotapi.PinChatMessageConfig{
				ChatID:              chatID,
				MessageID:           sentMsg.MessageID,
				DisableNotification: true,
			}
			if _, err := Bot.Request(pin); err != nil {
				log.Printf("PinChatMessage error: %v", err)
			}
		} else {
			log.Printf("Send auth success message error: %v", err)
		}
	default:
		sendMessage(chatID, "Неизвестный шаг авторизации. Напишите /start чтобы начать заново.")
	}
}

func completePayment(hostID, dealID int64, chatID int64) {
	o := orm.NewOrm()
	var host models.Host
	if err := o.QueryTable("host").Filter("Id", hostID).One(&host); err != nil {
		log.Printf("completePayment: host not found %d: %v", hostID, err)
		sendMessage(chatID, "❌ Воркер хоста не найден.")
		return
	}
	if host.PaymentMethodID == "" {
		log.Printf("completePayment: payment method not set for host %d", hostID)
		sendMessage(chatID, "❌ Метод оплаты не настроен.")
		return
	}
	url := fmt.Sprintf("https://app.cr.bot/internal/v1/p2c/payments/%d/complete", dealID)
	payload := map[string]string{"method": host.PaymentMethodID}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("completePayment: json marshal error: %v", err)
		sendMessage(chatID, "❌ Ошибка при формировании запроса.")
		return
	}
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Cookie", "access_token="+host.AccessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("completePayment: http error: %v", err)
		sendMessage(chatID, "❌ Ошибка при подтверждении сделки.")
		return
	}
	if resp != nil {
		resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Printf("completePayment: unexpected status %d", resp.StatusCode)
			// можно распарсить тело и показать пользователю причину
			sendMessage(chatID, "❌ Ошибка от сервера при подтверждении сделки.")
			return
		}
	}

	// Отправить подтверждение
	sendMessage(chatID, "✅ Сделка подтверждена!")
}

func sendMessage(chatID int64, text string) {
	if Bot == nil {
		log.Printf("Bot is nil, can't send message to %d: %s", chatID, text)
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := Bot.Send(msg); err != nil {
		log.Printf("sendMessage error to %d: %v", chatID, err)
	}
}
