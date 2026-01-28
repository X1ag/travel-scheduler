package telegram

import (
	"context"
	"log"

	"github.com/X1ag/TravelScheduler/internal/usecase"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Bot struct {
	client *bot.Bot
	tripUC *usecase.TripUsecase
	bookUC *usecase.BookUsecase
	userUC *usecase.UserUsecase
}

func NewBot(client *bot.Bot, tripUC *usecase.TripUsecase, bookUC *usecase.BookUsecase, userUC *usecase.UserUsecase) *Bot {
	return &Bot{
		client: client,
		tripUC: tripUC,
		bookUC: bookUC,
		userUC: userUC,
	}
}

func (b *Bot) Start(ctx context.Context) {
	b.client.Start(ctx)
}

func (b *Bot) AddClient(client *bot.Bot) {
	b.client = client
}

func DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Printf("Получено сообщение: %s", update.Message.Text)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Hello, world\\!",
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) SecondHandler(ctx context.Context, update *models.Update) {
	log.Printf("Получено сообщение: %s", update.Message.Text)
	_, err := b.client.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "dont write me\\!",
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}
