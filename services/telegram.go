package services

import (
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
		if update.Message == nil {
			continue
		}

		handleMessage(update.Message)
	}
}

func handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	mu.Lock()
	defer mu.Unlock()

	if text == "/start" {
		authStates[chatID] = &AuthState{Step: "awaiting_login"}
		sendMessage(chatID, "Введите ваш логин:")
		return
	}

	state, exists := authStates[chatID]
	if !exists {
		sendMessage(chatID, "Напишите /start для входа.")
		return
	}
	if strings.HasPrefix(text, "complete_") {
		parts := strings.Split(text, "_")
		if len(parts) == 3 {
			hostID, _ := strconv.ParseInt(parts[1], 10, 64)
			dealID, _ := strconv.ParseInt(parts[2], 10, 64)
			go completePayment(hostID, dealID, chatID)
		}
	}
	switch state.Step {
	case "awaiting_login":
		state.Login = text
		state.Step = "awaiting_password"
		sendMessage(chatID, "Введите ваш пароль:")

	case "awaiting_password":
		login := state.Login
		password := text
		delete(authStates, chatID) // очищаем состояние

		// Проверяем учётные данные
		o := orm.NewOrm()
		var user models.User
		err := o.QueryTable("user").Filter("Username", login).One(&user)
		if err != nil || user.Password != models.HashPassword(password) {
			sendMessage(chatID, "❌ Неверный логин или пароль.")
			return
		}

		// Обновляем пользователя: привязываем chat_id и генерируем токен
		user.TelegramChatID = chatID
		user.WebAppToken = utils.GenerateToken()
		o.Update(&user, "TelegramChatID", "WebAppToken")

		// Формируем Web App URL
		webAppURL := os.Getenv("WEB_APP_URL") + "/login/telegram?token=" + user.WebAppToken

		// Отправляем кнопку Web App
		btn := tgbotapi.NewInlineKeyboardButtonURL("Открыть админку", webAppURL)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(btn),
		)

		msg := tgbotapi.NewMessage(chatID, "Авторизация успешна!")
		msg.ReplyMarkup = keyboard
		sentMsg, _ := Bot.Send(msg)
		pin := tgbotapi.PinChatMessageConfig{
			ChatID:              chatID,
			MessageID:           sentMsg.MessageID,
			DisableNotification: true, // без уведомления
		}
		Bot.Request(pin)

	}
}
func completePayment(hostID, dealID int64, chatID int64) {
	o := orm.NewOrm()
	var host models.Host
	err := o.QueryTable("host").Filter("Id", hostID).One(&host)
	if err != nil {
		return
	}

	url := fmt.Sprintf("https://app.cr.bot/internal/v1/p2c/payments/%d/complete", dealID)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Cookie", "access_token="+host.AccessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, _ := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}

	// Отправить подтверждение
	sendMessage(chatID, "✅ Сделка подтверждена!")
}
func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	Bot.Send(msg)
}
