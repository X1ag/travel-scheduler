package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

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

func (b *Bot) RegisterUser(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	log.Printf("Получено сообщение: %s", update.Message.Text)
	
	user := &domain.User{
		TelegramID: update.Message.From.ID,
		Name:       update.Message.From.FirstName,
		Username:   update.Message.From.Username,
	}

	err := b.userUC.Create(ctx, user)
	if err != nil {
		err := sendErrorMessage(err, ctx, botClient, update)	
		if err != nil {
			log.Println(err)
		}
		return
	}
	_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("User is created with ID: %d",user.ID),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}

	log.Println("User created with id", user.ID)
}

func (b *Bot) GetSchedule(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	log.Printf("Получено сообщение: %s", update.Message.Text)

	log.Println("hello")
	splited := strings.Split(update.Message.Text, " ")	

	if len(splited) < 3 {
		err := sendErrorMessage(domain.ErrInvalidInput, ctx, botClient, update)	
		if err != nil {
			log.Println(err)
		}
		return
	}

	from := splited[1]
	to := splited[2]
	date := time.Now()

	if len(splited) == 4 {
		layout := "15:36:01"
		userTime := splited[4]	
		t, err := time.Parse(layout, userTime)
		if err != nil {
			fmt.Println("Error parsing time:", err)
			return
		}
		date = time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), t.Second(), 0, date.Location())
		log.Println("parsedTime is:", t)
		log.Println(date)
	}	

	options, err := b.tripUC.Search(ctx, from, to, date)
	if err != nil {
		err := sendErrorMessage(err, ctx, botClient, update)	
		if err != nil {
			log.Println(err)
		}
		return
	}
	resultText := "Рейсы \n"
	for _, opt := range options {
		resultText = resultText + fmt.Sprintf("%s %s %s\n", opt.Title, opt.DepartureTime, opt.TrainNumber)
	}
	log.Println("result text:", resultText)
	botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   resultText,
		ParseMode: models.ParseModeMarkdown,
	})
}

func (b *Bot) RegisterHandlers() {
    b.client.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, b.DefaultHandler)
    b.client.RegisterHandler(bot.HandlerTypeMessageText, "/createUser", bot.MatchTypeExact, b.RegisterUser)
    b.client.RegisterHandler(bot.HandlerTypeMessageText, "/getSchedule", bot.MatchTypeExact, b.GetSchedule)
}

func sendErrorMessage(err error, ctx context.Context, botClient *bot.Bot, update *models.Update) error {
	_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Error: %s", err.Error()),
			ParseMode: models.ParseModeMarkdown,
		})	
	if err != nil {
		return err
	}
	return nil
}