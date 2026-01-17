package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Item struct {
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
}

type ChatListData struct {
	Items           []Item `json:"items"`
	MessageID       int    `json:"message_id"`
	MessageThreadID int    `json:"message_thread_id"`
	ChatID          int64  `json:"chat_id"`
	Locked          bool   `json:"locked"`
}

func main() {
	redisClient = createRedisClient()
	defer redisClient.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithMessageTextHandler("/make_list", bot.MatchTypeExact, makeListCommand),
		bot.WithCallbackQueryDataHandler("btn_", bot.MatchTypePrefix, callbackHandler),
	}

	b, err := bot.New(os.Getenv("TELEGRAM_BOT_TOKEN"), opts...)
	if nil != err {
		panic(err)
	}

	b.Start(ctx)
}

func generateChatKey(message *models.Message) string {
	return strconv.Itoa(int(message.Chat.ID)) + ":" + strconv.Itoa(int(message.MessageThreadID))
}

func makeListCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	var chatListData ChatListData
	message, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          update.Message.Chat.ID,
		MessageThreadID: update.Message.MessageThreadID,
		Text:            "🗒",
		ReplyMarkup:     formatListDataButton(&chatListData),
	})
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

	var chatListData ChatListData
	getValue(ctx, redisClient, generateChatKey(update.CallbackQuery.Message.Message), &chatListData)

	if strings.HasPrefix(update.CallbackQuery.Data, "btn_item") {
		tokens := strings.Split(update.CallbackQuery.Data, "_")
		index, _ := strconv.Atoi(tokens[len(tokens)-1])
		chatListData.Items[index].Enable = !chatListData.Items[index].Enable
	}

	switch update.CallbackQuery.Data {
	case "btn_empty_list":
		{
			chatListData.Items = nil
		}
	case "btn_refresh_list":
		{
			var newList []Item
			for _, item := range chatListData.Items {
				if !item.Enable {
					newList = append(newList, item)
				}
			}
			chatListData.Items = newList
		}
	case "btn_list_locked":
		{
			chatListData.Locked = !chatListData.Locked
		}

	}
	setValue(ctx, redisClient, generateChatKey(update.CallbackQuery.Message.Message), chatListData)
	b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: formatListDataButton(&chatListData),
	})
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
		message, _ := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: messageThreadID,
			Text:            "🗒",
			ReplyMarkup:     formatListDataButton(chatListData),
		})
		chatListData.MessageID = message.ID
	} else {
		b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   chatListData.MessageID,
			ReplyMarkup: formatListDataButton(chatListData),
		})
	}
}

func formatListDataButton(chatListData *ChatListData) models.ReplyMarkup {
	var keyboard [][]models.InlineKeyboardButton
	for item_index, item := range chatListData.Items {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{
				Text:         buttonText(item),
				CallbackData: "btn_item_" + strconv.Itoa(item_index),
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
	if update.Message != nil {
		var chatListData ChatListData
		getValue(ctx, redisClient, generateChatKey(update.Message), &chatListData)
		if !chatListData.Locked && chatListData.MessageID != 0 && chatListData.MessageThreadID == update.Message.MessageThreadID {
			chatListData.Items = parseShoppingList(chatListData.Items, update.Message.Text)
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.ID,
			})
			drawShoppingList(ctx, b, update.Message.Chat.ID, update.Message.MessageThreadID, &chatListData)
			setValue(ctx, redisClient, generateChatKey(update.Message), chatListData)
		}
	}
}

func parseShoppingList(shoppingList []Item, message string) []Item {
	for line := range strings.SplitSeq(message, "\n") {
		shoppingList = append(shoppingList, Item{
			Name:   line,
			Enable: false,
		})
	}
	return shoppingList
}
