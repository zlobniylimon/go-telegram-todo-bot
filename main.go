package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Item struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
}

type ChatListData struct {
	Items           []Item `json:"items"`
	MessageID       int    `json:"message_id"`
	MessageThreadID int    `json:"message_thread_id"`
	ChatID          int64  `json:"chat_id"`
	Locked          bool   `json:"locked"`
	NextItemID      int    `json:"next_item_id"`
}

func main() {
	token := getRequiredEnv("TELEGRAM_BOT_TOKEN")

	redisClient = createRedisClient()
	defer redisClient.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithMessageTextHandler("/make_list", bot.MatchTypeExact, makeListCommand),
		bot.WithCallbackQueryDataHandler("btn_", bot.MatchTypePrefix, callbackHandler),
	}

	b, err := bot.New(token, opts...)
	if nil != err {
		panic(err)
	}

	b.Start(ctx)
}

func getRequiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("environment variable %s is required", name)
	}
	return value
}

func generateChatKey(message *models.Message) string {
	return strconv.Itoa(int(message.Chat.ID)) + ":" + strconv.Itoa(int(message.MessageThreadID))
}

func makeListCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	defer lockChat(generateChatKey(update.Message))()

	message, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          update.Message.Chat.ID,
		MessageThreadID: update.Message.MessageThreadID,
		Text:            "🗒",
		ReplyMarkup:     formatListDataButton(&ChatListData{}),
	})
	if err != nil {
		log.Printf("makeListCommand: send message: %v", err)
		return
	}

	var chatListData ChatListData
	chatListData.MessageID = message.ID
	chatListData.MessageThreadID = message.MessageThreadID
	chatListData.ChatID = message.Chat.ID
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: update.Message.ID,
	})
	setValue(ctx, redisClient, generateChatKey(update.Message), chatListData)
}

func callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	message := update.CallbackQuery.Message.Message
	defer lockChat(generateChatKey(message))()

	var chatListData ChatListData
	getValue(ctx, redisClient, generateChatKey(message), &chatListData)
	ensureItemIDs(&chatListData)

	if chatListData.MessageID != 0 && chatListData.MessageID != message.ID {
		return
	}

	applyCallback(&chatListData, update.CallbackQuery.Data)
	setValue(ctx, redisClient, generateChatKey(message), chatListData)
	b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      message.Chat.ID,
		MessageID:   message.ID,
		ReplyMarkup: formatListDataButton(&chatListData),
	})
}

func applyCallback(chatListData *ChatListData, callbackData string) {
	if strings.HasPrefix(callbackData, "btn_item_") {
		id, err := strconv.ParseInt(strings.TrimPrefix(callbackData, "btn_item_"), 10, 64)
		if err != nil {
			return
		}
		for i := range chatListData.Items {
			if chatListData.Items[i].ID == int(id) {
				chatListData.Items[i].Enable = !chatListData.Items[i].Enable
				return
			}
		}
		return
	}

	switch callbackData {
	case "btn_empty_list":
		chatListData.Items = nil
	case "btn_refresh_list":
		var newList []Item
		for _, item := range chatListData.Items {
			if !item.Enable {
				newList = append(newList, item)
			}
		}
		chatListData.Items = newList
	case "btn_list_locked":
		chatListData.Locked = !chatListData.Locked
	}
}

func ensureItemIDs(chatListData *ChatListData) {
	for i := range chatListData.Items {
		if chatListData.Items[i].ID == 0 {
			chatListData.NextItemID++
			chatListData.Items[i].ID = chatListData.NextItemID
		}
		if chatListData.Items[i].ID > chatListData.NextItemID {
			chatListData.NextItemID = chatListData.Items[i].ID
		}
	}
}

func buttonText(item Item) string {
	if item.Enable {
		return "✅ " + item.Name
	}

	return "❌ " + item.Name
}

func lockedImage(chatListData *ChatListData) string {
	if chatListData.Locked {
		return "🔒"
	}
	return "🔓"
}

func drawShoppingList(ctx context.Context, b *bot.Bot, chatID int64, messageThreadID int, chatListData *ChatListData) {
	if chatListData.MessageID == 0 {
		message, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: messageThreadID,
			Text:            "🗒",
			ReplyMarkup:     formatListDataButton(chatListData),
		})
		if err != nil {
			log.Printf("drawShoppingList: send message: %v", err)
			return
		}
		chatListData.MessageID = message.ID
		return
	}

	_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      chatID,
		MessageID:   chatListData.MessageID,
		ReplyMarkup: formatListDataButton(chatListData),
	})
	if err != nil {
		log.Printf("drawShoppingList: edit message reply markup: %v", err)
	}
}

func formatListDataButton(chatListData *ChatListData) models.ReplyMarkup {
	var keyboard [][]models.InlineKeyboardButton
	for _, item := range chatListData.Items {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{
				Text:         buttonText(item),
				CallbackData: "btn_item_" + strconv.Itoa(item.ID),
			},
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{
			Text:         "🗑",
			CallbackData: "btn_empty_list",
		},
		{
			Text:         "🔄",
			CallbackData: "btn_refresh_list",
		},
	})

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{
			Text:         lockedImage(chatListData),
			CallbackData: "btn_list_locked",
		},
	})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	return kb
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	defer lockChat(generateChatKey(update.Message))()

	var chatListData ChatListData
	getValue(ctx, redisClient, generateChatKey(update.Message), &chatListData)
	if !chatListData.Locked && chatListData.MessageID != 0 && chatListData.MessageThreadID == update.Message.MessageThreadID {
		ensureItemIDs(&chatListData)
		parseShoppingList(&chatListData, update.Message.Text)
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: update.Message.ID,
		})
		drawShoppingList(ctx, b, update.Message.Chat.ID, update.Message.MessageThreadID, &chatListData)
		setValue(ctx, redisClient, generateChatKey(update.Message), chatListData)
	}
}

func parseShoppingList(chatListData *ChatListData, message string) {
	for line := range strings.SplitSeq(message, "\n") {
		chatListData.NextItemID++
		chatListData.Items = append(chatListData.Items, Item{
			ID:     chatListData.NextItemID,
			Name:   line,
			Enable: false,
		})
	}
}
