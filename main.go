package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var botToken = "8441638008:AAFjrF8PQtHfWmX3AaqThjF7cYt_4d3CPbQ"
var apiURL = "https://api.telegram.org/bot" + botToken

// Настройки MTProto прокси из Telegram Desktop
var proxyServer = "127.0.0.1:1443" // замените на ваш хост:порт

var client *http.Client

func main() {
	// Создаём HTTP клиент, который ходит через SOCKS5
	proxyURL, _ := url.Parse("socks5://" + proxyServer)
	
	client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 30 * time.Second,
	}

	fmt.Println("Проверяю подключение...")
	resp, err := client.Get("https://api.telegram.org")
	if err != nil {
		fmt.Println("❌ Ошибка подключения:", err)
		fmt.Println("\nВозможно, нужен другой адрес прокси.")
		fmt.Println("Откройте Telegram Desktop → Настройки → Прокси")
		fmt.Println("Найдите SOCKS5 или HTTP прокси (не MTProto)")
		return
	}
	resp.Body.Close()
	fmt.Println("✅ Бот запущен!")
	
	// Запускаем бота
	runBot()
}

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  Message  `json:"message"`
	Callback Callback `json:"callback_query"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      Chat   `json:"chat"`
	From      User   `json:"from"`
}

type Callback struct {
	ID      string  `json:"id"`
	Data    string  `json:"data"`
	Message Message `json:"message"`
	From    User    `json:"from"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

func runBot() {
	offset := 0
	for {
		updates, err := getUpdates(offset)
		if err != nil {
			log.Println("Ошибка:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Callback.ID != "" {
				handleCallback(update.Callback)
			} else if update.Message.Text != "" {
				handleMessage(update.Message)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func getUpdates(offset int) ([]Update, error) {
	url := apiURL + "/getUpdates?offset=" + strconv.Itoa(offset) + "&timeout=10"
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result []Update `json:"result"`
	}
	json.Unmarshal(body, &result)
	return result.Result, nil
}

func handleMessage(msg Message) {
	switch msg.Text {
	case "/start":
		text := fmt.Sprintf("👋 Привет, %s!\n\nДобро пожаловать в SuperVPN!\n/profile — профиль\n/buy — купить подписку", msg.From.FirstName)
		sendMessage(msg.Chat.ID, text, nil)

	case "/profile":
		text := fmt.Sprintf("👤 %s | 🥉 Бронза\n├ ID: %d\n├ С нами: Сегодня\n└ Баланс: 0.00 ₽\n\n📱 Подписка 🔴 Неактивна\n├ Тариф: Не выбран\n└ Нажми /buy чтобы активировать\n\n👥 Рефералы: 0\n📊 Трафик: 0 B", msg.From.FirstName, msg.From.ID)
		keyboard := map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{{"text": "💰 Пополнить", "callback_data": "balance_topup"}, {"text": "📱 Устройства", "callback_data": "devices_list"}},
				{{"text": "👥 Рефералы", "callback_data": "referral_info"}, {"text": "📊 Статистика", "callback_data": "traffic_stats"}},
				{{"text": "💳 История", "callback_data": "transaction_history"}, {"text": "⚙️ Ещё", "callback_data": "more_options"}},
			},
		}
		sendMessage(msg.Chat.ID, text, keyboard)

	case "/buy":
		keyboard := map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{{"text": "📅 Месяц — 299 ₽", "callback_data": "buy_month"}},
				{{"text": "📅 Год — 1990 ₽", "callback_data": "buy_year"}},
				{{"text": "🎁 Пробный (3 дня)", "callback_data": "buy_trial"}},
				{{"text": "◀️ Назад", "callback_data": "back_to_profile"}},
			},
		}
		sendMessage(msg.Chat.ID, "🛒 Выберите тариф:", keyboard)

	default:
		sendMessage(msg.Chat.ID, "Используй /profile или /buy", nil)
	}
}

func handleCallback(cb Callback) {
	chatID := cb.Message.Chat.ID
	switch cb.Data {
	case "back_to_profile":
		editOrSend(chatID, cb.Message.MessageID, "Напиши /profile", nil)
	case "buy_trial":
		answerCallback(cb.ID, "🎁 Пробный период активирован!")
		editOrSend(chatID, cb.Message.MessageID, "✅ Пробный период на 3 дня активирован!", nil)
	case "buy_month":
		answerCallback(cb.ID, "✅ Тариф Месяц выбран!")
	case "buy_year":
		answerCallback(cb.ID, "✅ Тариф Год выбран!")
	case "devices_list":
		keyboard := map[string]interface{}{"inline_keyboard": [][]map[string]string{{{"text": "◀️ Назад", "callback_data": "back_to_profile"}}}}
		editOrSend(chatID, cb.Message.MessageID, "📱 Устройства\n\nПока нет подключённых устройств.", keyboard)
	case "referral_info":
		keyboard := map[string]interface{}{"inline_keyboard": [][]map[string]string{{{"text": "📋 Копировать ссылку", "callback_data": "copy_ref_link"}}, {{"text": "◀️ Назад", "callback_data": "back_to_profile"}}}}
		editOrSend(chatID, cb.Message.MessageID, "👥 Рефералы\n\nПриглашено: 0 | Активных: 0 | Заработано: 0 ₽", keyboard)
	case "traffic_stats":
		keyboard := map[string]interface{}{"inline_keyboard": [][]map[string]string{{{"text": "◀️ Назад", "callback_data": "back_to_profile"}}}}
		editOrSend(chatID, cb.Message.MessageID, "📊 Трафик\n\nСегодня: 0 B | Неделя: 0 B | Месяц: 0 B", keyboard)
	case "more_options":
		keyboard := map[string]interface{}{"inline_keyboard": [][]map[string]string{{{"text": "🔄 Продлить", "callback_data": "renew"}}, {{"text": "🎁 Пробный", "callback_data": "buy_trial"}}, {{"text": "💬 Поддержка", "callback_data": "support"}}, {{"text": "◀️ Назад", "callback_data": "back_to_profile"}}}}
		editOrSend(chatID, cb.Message.MessageID, "⚙️ Дополнительно", keyboard)
	default:
		answerCallback(cb.ID, "Нажато: "+cb.Data)
	}
}

func sendMessage(chatID int64, text string, replyMarkup map[string]interface{}) {
	data := map[string]interface{}{"chat_id": chatID, "text": text, "parse_mode": "HTML"}
	if replyMarkup != nil { data["reply_markup"] = replyMarkup }
	jsonData, _ := json.Marshal(data)
	client.Post(apiURL+"/sendMessage", "application/json", bytes.NewBuffer(jsonData))
}

func editOrSend(chatID int64, messageID int, text string, replyMarkup map[string]interface{}) {
	data := map[string]interface{}{"chat_id": chatID, "message_id": messageID, "text": text, "parse_mode": "HTML"}
	if replyMarkup != nil { data["reply_markup"] = replyMarkup }
	jsonData, _ := json.Marshal(data)
	resp, err := client.Post(apiURL+"/editMessageText", "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp.StatusCode != 200 { sendMessage(chatID, text, replyMarkup) }
}

func answerCallback(callbackID string, text string) {
	data := map[string]string{"callback_query_id": callbackID, "text": text}
	jsonData, _ := json.Marshal(data)
	client.Post(apiURL+"/answerCallbackQuery", "application/json", bytes.NewBuffer(jsonData))
}