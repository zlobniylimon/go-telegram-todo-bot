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
	Bought bool   `json:"bought"`
}

type ChatData struct {
	Items           []Item `json:"items"`
	MessageID       int    `json:"message_id"`
	MessageThreadID int    `json:"message_thread_id"`
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

func makeListCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	var chatData ChatData
	message, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          update.Message.Chat.ID,
		MessageThreadID: update.Message.MessageThreadID,
		Text:            "ToDo List",
		ReplyMarkup:     formatItemsIntoButton(chatData.Items),
	})
	chatData.MessageID = message.ID
	chatData.MessageThreadID = message.MessageThreadID

	setValue(ctx, redisClient, strconv.Itoa(int(update.Message.Chat.ID)), chatData)
}

func callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	var chatData ChatData
	getValue(ctx, redisClient, strconv.Itoa(int(update.CallbackQuery.Message.Message.Chat.ID)), &chatData)

	if strings.HasPrefix(update.CallbackQuery.Data, "btn_item") {
		tokens := strings.Split(update.CallbackQuery.Data, "_")
		index, _ := strconv.Atoi(tokens[len(tokens)-1])
		chatData.Items[index].Bought = !chatData.Items[index].Bought
	}

	switch update.CallbackQuery.Data {
	case "btn_empty_list":
		{
			chatData.Items = nil
		}
	case "btn_refresh_list":
		{
			var newList []Item
			for _, item := range chatData.Items {
				if !item.Bought {
					newList = append(newList, item)
				}
			}
			chatData.Items = newList
		}
	}
	setValue(ctx, redisClient, strconv.Itoa(int(update.CallbackQuery.Message.Message.Chat.ID)), chatData)
	b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: formatItemsIntoButton(chatData.Items),
	})
}

func buttonText(item Item) string {
	if item.Bought {
		return "✅ " + item.Name
	}

	return "❌ " + item.Name
}

func drawShoppingList(ctx context.Context, b *bot.Bot, chatID int64, messageThreadID int, chatData *ChatData) {
	if chatData.MessageID == 0 {
		message, _ := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: messageThreadID,
			Text:            "ToDo List",
			ReplyMarkup:     formatItemsIntoButton(chatData.Items),
		})
		chatData.MessageID = message.ID
	} else {
		b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   chatData.MessageID,
			ReplyMarkup: formatItemsIntoButton(chatData.Items),
		})
	}
}

func formatItemsIntoButton(items []Item) models.ReplyMarkup {
	var keyboard [][]models.InlineKeyboardButton
	for item_index, item := range items {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{
				Text:         buttonText(item),
				CallbackData: "btn_item_" + strconv.Itoa(item_index),
			},
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{
			Text:         "empty list",
			CallbackData: "btn_empty_list",
		},
		{
			Text:         "refresh list",
			CallbackData: "btn_refresh_list",
		},
	})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	return kb
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		var chatData ChatData
		getValue(ctx, redisClient, strconv.Itoa(int(update.Message.Chat.ID)), &chatData)
		if chatData.MessageID != 0 && chatData.MessageThreadID == update.Message.MessageThreadID {
			chatData.Items = parseShoppingList(chatData.Items, update.Message.Text)
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.ID,
			})
			drawShoppingList(ctx, b, update.Message.Chat.ID, update.Message.MessageThreadID, &chatData)
			setValue(ctx, redisClient, strconv.Itoa(int(update.Message.Chat.ID)), chatData)
		}
	}
}

func parseShoppingList(shoppingList []Item, message string) []Item {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		shoppingList = append(shoppingList, Item{
			Name:   line,
			Bought: false,
		})
	}
	return shoppingList
}
