package telegram

import (
	"context"
	"fmt"
	"log"

	"github.com/X1ag/TravelScheduler/internal/domain"
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

func (b *Bot) AddClient(botClient *bot.Bot) {
	b.client = botClient
}

func (b *Bot) DefaultHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	log.Printf("Получено сообщение: %s", update.Message.Text)
	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Hello, world\\!",
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) SecondHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	log.Printf("Получено сообщение: %s", update.Message.Text)
	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "i try to reg user, wait\\!",
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
	user := &domain.User{
		TelegramID: update.Message.From.ID,
		Name:       update.Message.From.FirstName,
		Username:   update.Message.From.Username,
	}

	err = b.userUC.Create(ctx, user)
	if err != nil {
		_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Error: %s", err.Error()),
			ParseMode: models.ParseModeMarkdown,
		})	
	}
	_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("user is created\\! ID: %d",user.ID),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
	log.Println("User created with id", user.ID)
}

func (b *Bot) RegisterHandlers() {
    // Регистрируем конкретные команды
    b.client.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, b.DefaultHandler)
    
    b.client.RegisterHandler(bot.HandlerTypeMessageText, "/createUser", bot.MatchTypeExact, b.SecondHandler)
}
