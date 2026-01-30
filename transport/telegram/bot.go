package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/X1ag/TravelScheduler/internal/domain"
	"github.com/X1ag/TravelScheduler/internal/usecase"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type UserState string

const (
	StateNone        UserState = "none"
	StateWaitingFrom UserState = "waiting_from"
	StateWaitingTo   UserState = "waiting_to"
	StateShowingSchedule UserState = "showing_schedule"
)

type UserSession struct {
	State    UserState
	From     string
	To       string
	Date     time.Time
	Schedule []*domain.Schedule
}

type Bot struct {
	client      *bot.Bot
	tripUC      *usecase.TripUsecase
	bookUC      *usecase.BookUsecase
	userUC      *usecase.UserUsecase
	userSessions map[int64]*UserSession // telegramID -> session
	mu          sync.RWMutex
}

func NewBot(client *bot.Bot, tripUC *usecase.TripUsecase, bookUC *usecase.BookUsecase, userUC *usecase.UserUsecase) *Bot {
	return &Bot{
		client:       client,
		tripUC:       tripUC,
		bookUC:       bookUC,
		userUC:       userUC,
		userSessions: make(map[int64]*UserSession),
	}
}

func (b *Bot) getSession(telegramID int64) *UserSession {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if session, exists := b.userSessions[telegramID]; exists {
		return session
	}
	
	session := &UserSession{
		State: StateNone,
		Date:  time.Now(),
	}
	b.userSessions[telegramID] = session
	return session
}

func (b *Bot) clearSession(telegramID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.userSessions, telegramID)
}

func (b *Bot) ensureUser(ctx context.Context, telegramID int64, firstName, username string) (*domain.User, error) {
	user, err := b.userUC.GetUserByTelegramID(ctx, telegramID)
	if err == nil {
		return user, nil
	}
	
	newUser := &domain.User{
		TelegramID: telegramID,
		Name:       firstName,
		Username:   username,
	}
	
	err = b.userUC.Create(ctx, newUser)
	if err != nil {
		return nil, err
	}
	
	return newUser, nil
}

func (b *Bot) Start(ctx context.Context) {
	b.client.Start(ctx)
}

func (b *Bot) AddClient(botClient *bot.Bot) {
	b.client = botClient
}

func (b *Bot) StartHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	
	_, err := b.ensureUser(ctx, telegramID, update.Message.From.FirstName, update.Message.From.Username)
	if err != nil {
		log.Printf("Ошибка регистрации пользователя: %v", err)
		sendErrorMessage(err, ctx, botClient, update)
		return
	}
	
	welcomeText := "👋 *Добро пожаловать в TravelPet\\!*\n\n" +
		"Я помогу вам планировать поездки и напомню о них заранее\\.\n\n" +
		"*Доступные команды:*\n" +
		"/newtrip — создать новую поездку\n" +
		"/mytrips — мои поездки\n" +
		"/help — справка\n\n" +
		"Начнем планировать поездку? Нажмите /newtrip"

	_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      welcomeText,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) HelpHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	helpText := "📖 *Справка по командам*\n\n" +
		"/newtrip — создать новую поездку\n" +
		"   Бот проведет вас через пошаговый процесс создания поездки\n\n" +
		"/mytrips — показать все ваши запланированные поездки\n\n" +
		"/help — показать эту справку\n\n" +
		"*Как создать поездку:*\n" +
		"1\\. Нажмите /newtrip\n" +
		"2\\. Введите станцию отправления \\(например: s9613483 или Таганрог\\)\n" +
		"3\\. Введите станцию назначения\n" +
		"4\\. Выберите поезд из предложенного расписания\n" +
		"5\\. Готово\\! Бот напомнит вам за 30 минут до отправления"

	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      helpText,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) NewTripHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID

	if b.userSessions[telegramID] == nil {
		_, err := b.ensureUser(ctx, telegramID, update.Message.From.FirstName, update.Message.From.Username)
		if err != nil {
			log.Printf("Ошибка регистрации пользователя: %v", err)
		}
	}
	
	session := b.getSession(telegramID)
	session.State = StateWaitingFrom
	session.From = ""
	session.To = ""
	session.Date = time.Now()
	session.Schedule = nil
	
	text := "🚆 *Создание новой поездки*\n\n" +
		"Шаг 1 из 3: *Введите станцию отправления*\n\n" +
		"Вы можете ввести:\n" +
		"• Код станции \\(например: s9613483\\)\n" +
		"• Название станции \\(например: Таганрог\\)\n\n" +
		"_Для отмены введите /cancel_"

	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) MyTripsHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	
	user, err := b.userUC.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		sendErrorMessage(err, ctx, botClient, update)
		return
	}
	
	trips, err := b.tripUC.GetByUserID(ctx, user.ID)
	if err != nil {
		sendErrorMessage(err, ctx, botClient, update)
		return
	}
	
	if len(trips) == 0 {
		text := "📋 *Мои поездки*\n\n" +
			"У вас пока нет запланированных поездок\\.\n\n" +
			"Создайте новую поездку командой /newtrip"

		_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      text,
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			log.Println(err)
		}
		return
	}
	
	var sb strings.Builder
	sb.WriteString("📋 *Мои поездки*\n\n")
	
	for i, trip := range trips {
		depTime := trip.DepartureTime.Format("02.01.2006 15:04")
		escapedFrom := escapeMarkdown(trip.From)
		escapedTo := escapeMarkdown(trip.To)
		sb.WriteString(fmt.Sprintf("*%d\\.* 🚆 Поездка #%d\n", i+1, trip.ID))
		sb.WriteString(fmt.Sprintf("   📍 %s → %s\n", escapedFrom, escapedTo))
		sb.WriteString(fmt.Sprintf("   🕒 %s\n\n", depTime))
	}
	
	_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      sb.String(),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) CancelHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	b.clearSession(telegramID)
	
	text := "❌ *Создание поездки отменено*\n\n" +
		"Вы можете начать заново командой /newtrip"

	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) TextMessageHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	session := b.getSession(telegramID)
	text := strings.TrimSpace(update.Message.Text)
	
	if text == "/cancel" || text == "/cancel_" {
		b.CancelHandler(ctx, botClient, update)
		return
	}
	
	switch session.State {
	case StateWaitingFrom:
		session.From = text
		session.State = StateWaitingTo
		
		escapedText := escapeMarkdown(text)
		msgText := fmt.Sprintf("✅ Станция отправления: *%s*\n\n"+
			"Шаг 2 из 3: *Введите станцию назначения*\n\n"+
			"Вы можете ввести:\n"+
			"• Код станции \\(например: s9612913\\)\n"+
			"• Название станции \\(например: Ростов\\-на\\-Дону\\)\n\n"+
			"_Для отмены введите /cancel_", escapedText)
		
		_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      msgText,
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			log.Println(err)
		}
		
	case StateWaitingTo:
		session.To = text
		session.State = StateShowingSchedule
		
		options, err := b.tripUC.Search(ctx, session.From, session.To, session.Date)
		if err != nil {
			session.State = StateWaitingTo
			sendErrorMessage(err, ctx, botClient, update)
			return
		}
		
		if len(options) == 0 {
			session.State = StateWaitingTo
			_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❌ Рейсы не найдены. Попробуйте другие станции или введите /cancel для отмены.",
			})
			if err != nil {
				log.Println(err)
			}
			return
		}
		
		b.sendScheduleWithButtons(ctx, botClient, update, options, session)
		
	default:
		_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Для начала создания поездки используйте команду /newtrip\nДля справки: /help",
		})
		if err != nil {
			log.Println(err)
		}
	}
}

func (b *Bot) sendScheduleWithButtons(ctx context.Context, botClient *bot.Bot, update *models.Update, options []*domain.Schedule, session *UserSession) {
	text := buildScheduleText(options, session.From, session.To)
	
	session.Schedule = options
	
	var buttons [][]models.InlineKeyboardButton
	for i := range options {
		callbackData := fmt.Sprintf("train:%d", i)
		
		opt := options[i]
		depTime := opt.DepartureTime.Format("15:04")
		buttonText := fmt.Sprintf("🚆 %s → %s", depTime, opt.ArrivalTime.Format("15:04"))
		
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text:         buttonText,
				CallbackData: callbackData,
			},
		})
	}
	
	buttons = append(buttons, []models.InlineKeyboardButton{
		{
			Text:         "❌ Отменить",
			CallbackData: "cancel",
		},
	})
	
	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) CallbackQueryHandler(ctx context.Context, botClient *bot.Bot, update *models.Update) {
	callbackQuery := update.CallbackQuery
	if callbackQuery == nil || callbackQuery.Data == "" {
		return
	}
	
	telegramID := callbackQuery.From.ID
	session := b.getSession(telegramID)
	
	callbackData := callbackQuery.Data
	
	if strings.HasPrefix(callbackData, "train:") {
		indexStr := strings.TrimPrefix(callbackData, "train:")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			log.Printf("Ошибка парсинга индекса поезда: %v", err)
			return
		}
		
		if session.Schedule == nil || index < 0 || index >= len(session.Schedule) {
			sendCallbackError(ctx, botClient, callbackQuery, "Ошибка: расписание не найдено")
			return
		}
		
		opt := session.Schedule[index]
		trainID := opt.TrainID
		departureTime := opt.DepartureTime
		
		user, err := b.userUC.GetUserByTelegramID(ctx, telegramID)
		if err != nil {
			sendCallbackError(ctx, botClient, callbackQuery, "Ошибка получения данных пользователя")
			return
		}
		
		tr := &domain.Trip{
			UserID:        user.ID,
			From:          session.From,
			To:            session.To,
			DepartureTime: departureTime,
		}
		
		err = b.tripUC.ConfirmTrip(ctx, tr)
		if err != nil {
			sendCallbackError(ctx, botClient, callbackQuery, fmt.Sprintf("Ошибка создания поездки: %s", err.Error()))
			return
		}
		
		b.clearSession(telegramID)
		
		depTime := departureTime.Format("02.01.2006 15:04")
		escapedTrainID := escapeMarkdown(trainID)
		escapedFrom := escapeMarkdown(session.From)
		escapedTo := escapeMarkdown(session.To)
		successText := fmt.Sprintf("✅ *Поездка успешно создана\\!*\n\n"+
			"📋 *Детали поездки:*\n"+
			"🚆 Поезд: *%s*\n"+
			"📍 Маршрут: *%s* → *%s*\n"+
			"🕒 Отправление: *%s*\n\n"+
			"Я напомню вам за 30 минут до отправления\\. Приятной поездки\\! 🚂",
			escapedTrainID, escapedFrom, escapedTo, depTime)
		
		var chatID int64
		var messageID int
		if callbackQuery.Message.Message != nil {
			msg := callbackQuery.Message.Message
			chatID = msg.Chat.ID
			messageID = msg.ID
			_, err = botClient.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      successText,
			})
			if err == nil {
				_, _ = botClient.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: callbackQuery.ID,
					Text:            "Поездка создана!",
				})
				return
			}
		}
		
		if chatID == 0 {
			chatID = callbackQuery.From.ID
		}
		_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      successText,
		})
		if err != nil {
			log.Println(err)
		}
		
		_, _ = botClient.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQuery.ID,
			Text:            "Поездка создана!",
		})
		
		return
	}
	
	if callbackData == "cancel" {
		b.clearSession(telegramID)
		
		var chatID int64
		var messageID int
		if callbackQuery.Message.Message != nil {
			msg := callbackQuery.Message.Message
			chatID = msg.Chat.ID
			messageID = msg.ID
			_, err := botClient.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      "❌ *Создание поездки отменено*\n\nВы можете начать заново командой /newtrip",
			})
			if err == nil {
				_, _ = botClient.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: callbackQuery.ID,
					Text:            "Отменено",
				})
				return
			}
		}
		
		if chatID == 0 {
			chatID = callbackQuery.From.ID
		}
		_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      "❌ *Создание поездки отменено*\n\nВы можете начать заново командой /newtrip",
		})
		if err != nil {
			log.Println(err)
		}
		
		_, _ = botClient.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQuery.ID,
			Text:            "Отменено",
		})
	}
}

func sendCallbackError(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, message string) {
	_, _ = botClient.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackQuery.ID,
		Text:            message,
		ShowAlert:       true,
	})
}

func (b *Bot) RegisterHandlers() {
	b.client.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, b.StartHandler)
	b.client.RegisterHandler(bot.HandlerTypeMessageText, "/newtrip", bot.MatchTypeExact, b.NewTripHandler)
	b.client.RegisterHandler(bot.HandlerTypeMessageText, "/mytrips", bot.MatchTypeExact, b.MyTripsHandler)
	b.client.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, b.HelpHandler)
	b.client.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, b.CancelHandler)
	b.client.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, b.TextMessageHandler)
	b.client.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, b.CallbackQueryHandler)
}

func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string) {
	_, err := b.client.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
	})
	if err != nil {
		log.Println(err)
	}
}

func sendErrorMessage(err error, ctx context.Context, botClient *bot.Bot, update *models.Update) error {
	_, err = botClient.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Error: %s", err.Error()),
		})	
	if err != nil {
		return err
	}
	return nil
}

func humanDurationFromSeconds(sec int) string {
	if sec < 60 {
		return fmt.Sprintf("%d сек", sec)
	}
	mins := sec / 60
	if mins < 60 {
		return fmt.Sprintf("%d мин", mins)
	}
	h := mins / 60
	m := mins % 60
	if m == 0 {
		return fmt.Sprintf("%dч (%d мин)", h, mins)
	}
	return fmt.Sprintf("%dч %dм (%d мин)", h, m, mins)
}

func cleanTitle(s string) string {
	s = strings.ReplaceAll(s, "\\", "")
	s = strings.ReplaceAll(s, "*", "")
	return s
}

func escapeMarkdown(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "_", "\\_")
	text = strings.ReplaceAll(text, "*", "\\*")
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	text = strings.ReplaceAll(text, "~", "\\~")
	text = strings.ReplaceAll(text, "`", "\\`")
	text = strings.ReplaceAll(text, ">", "\\>")
	text = strings.ReplaceAll(text, "#", "\\#")
	text = strings.ReplaceAll(text, "+", "\\+")
	text = strings.ReplaceAll(text, "-", "\\-")
	text = strings.ReplaceAll(text, "=", "\\=")
	text = strings.ReplaceAll(text, "|", "\\|")
	text = strings.ReplaceAll(text, "{", "\\{")
	text = strings.ReplaceAll(text, "}", "\\}")
	text = strings.ReplaceAll(text, ".", "\\.")
	text = strings.ReplaceAll(text, "!", "\\!")
	return text
}

func buildScheduleText(options []*domain.Schedule, from, to string) string {
	var b strings.Builder
	b.WriteString("🚆 *Расписание рейсов*\n\n")
	escapedFrom := escapeMarkdown(from)
	escapedTo := escapeMarkdown(to)
	b.WriteString(fmt.Sprintf("📍 *%s* → *%s*\n\n", escapedFrom, escapedTo))
	b.WriteString("Выберите поезд:\n\n")

	for i, opt := range options {
		num := i + 1
		title := escapeMarkdown(cleanTitle(opt.Title))
		escapedTrainID := escapeMarkdown(opt.TrainID)

		dep := opt.DepartureTime.Format("02.01.2006 15:04")
		arr := opt.ArrivalTime.Format("15:04")
		durationStr := humanDurationFromSeconds(int(opt.Duration))

		b.WriteString(fmt.Sprintf("*%d\\.* %s\n", num, title))
		b.WriteString(fmt.Sprintf("   🚆 Поезд: `%s`\n", escapedTrainID))
		b.WriteString(fmt.Sprintf("   🕒 %s → %s\n", dep, arr))
		b.WriteString(fmt.Sprintf("   ⏱ %s\n\n", durationStr))
	}

	return b.String()
}