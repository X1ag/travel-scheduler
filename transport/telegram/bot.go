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
	"github.com/X1ag/TravelScheduler/internal/utils"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type UserState string

const (
	StateNone            UserState = "none"
	StateSelectingFrom   UserState = "selecting_from"     // Inline station buttons
	StateSelectingTo     UserState = "selecting_to"       // Inline station buttons
	StateShowingSchedule UserState = "showing_schedule"   // Paginated results
	// Legacy states for backward compatibility during migration
	StateWaitingFrom UserState = "waiting_from"
	StateWaitingTo   UserState = "waiting_to"
)

type UserSession struct {
	State        UserState
	StateHistory []UserState // For back navigation

	From     string // Station code
	FromName string // Display name
	To       string // Station code
	ToName   string // Display name
	Date     time.Time
	Schedule []*domain.Schedule // Full schedule (not limited to 5)

	SchedulePage   int              // Current page for pagination
	RecentStations []utils.StationOption  // Last 5 used stations
	LastMessageID  int              // For editing messages
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

// ParseCallback parses callback data into action and parameters
// Format: action:param1:param2
func ParseCallback(data string) (action string, params []string) {
	parts := strings.Split(data, ":")
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// transitionState adds new state to history and updates current state
func (b *Bot) transitionState(session *UserSession, newState UserState) {
	session.StateHistory = append(session.StateHistory, newState)
	session.State = newState
}

// addToRecentStations adds a station to the recent stations list
// Maintains a max of 5 recent stations with most recent first
func (b *Bot) addToRecentStations(session *UserSession, station utils.StationOption) {
	// Remove if already exists (dedup)
	for i, s := range session.RecentStations {
		if s.Code == station.Code {
			session.RecentStations = append(session.RecentStations[:i], session.RecentStations[i+1:]...)
			break
		}
	}

	// Add to front
	session.RecentStations = append([]utils.StationOption{station}, session.RecentStations...)

	// Keep only last 5
	if len(session.RecentStations) > 5 {
		session.RecentStations = session.RecentStations[:5]
	}
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
	// Reset session
	session.State = StateSelectingFrom
	session.StateHistory = []UserState{StateSelectingFrom}
	session.From = ""
	session.FromName = ""
	session.To = ""
	session.ToName = ""
	session.Date = time.Now()
	session.Schedule = nil
	session.SchedulePage = 0

	// Show inline station selection
	b.showStationSelection(ctx, botClient, update.Message.Chat.ID, session, "from")
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
		// Text input for "From" station
		session.From = text
		session.FromName = text // Use text as display name for now
		b.transitionState(session, StateWaitingTo)

		// Try to add to recent if it's a known station
		if station, found := utils.GetStationByCode(text); found {
			session.FromName = station.DisplayName
			b.addToRecentStations(session, station)
		}

		escapedText := escapeMarkdown(session.FromName)
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
		// Text input for "To" station
		session.To = text
		session.ToName = text // Use text as display name for now
		b.transitionState(session, StateShowingSchedule)

		// Try to add to recent if it's a known station
		if station, found := utils.GetStationByCode(text); found {
			session.ToName = station.DisplayName
			b.addToRecentStations(session, station)
		}

		options, err := b.tripUC.Search(ctx, session.From, session.To, session.Date)
		if err != nil {
			session.State = StateWaitingTo
			sendErrorMessage(err, ctx, botClient, update)
			return
		}

		if len(options) == 0 {
			session.State = StateWaitingTo
			b.sendRecoverableError(ctx, botClient, update.Message.Chat.ID,
				"Рейсы не найдены для этого маршрута.",
				[]models.InlineKeyboardButton{
					{Text: "🔄 Попробовать снова", CallbackData: "ef"},
					{Text: "❌ Отменить", CallbackData: "x"},
				})
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
	// Use display names if available, fallback to codes
	fromDisplay := session.FromName
	if fromDisplay == "" {
		fromDisplay = session.From
	}
	toDisplay := session.ToName
	if toDisplay == "" {
		toDisplay = session.To
	}

	text := buildScheduleText(options, fromDisplay, toDisplay)
	session.Schedule = options
	session.SchedulePage = 0

	// Use new pagination keyboard
	keyboard := b.buildScheduleKeyboard(options, session.SchedulePage)

	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		// No ParseMode - avoid markdown escaping issues
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
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

	action, params := ParseCallback(callbackQuery.Data)

	// Route to appropriate handler
	switch action {
	case "b": // Back
		b.handleBack(ctx, botClient, callbackQuery, session)

	case "x": // Cancel
		b.handleCancel(ctx, botClient, callbackQuery, session)

	case "ss": // Select Station
		b.handleSelectStation(ctx, botClient, callbackQuery, session, params)

	case "tr": // Select Train
		b.handleTrainSelect(ctx, botClient, callbackQuery, session, params)

	case "sp": // Schedule Page
		b.handleSchedulePage(ctx, botClient, callbackQuery, session, params)

	case "ef": // Edit From
		session.State = StateSelectingFrom
		b.transitionState(session, StateSelectingFrom)
		if callbackQuery.Message.Message != nil {
			b.showStationSelection(ctx, botClient, callbackQuery.Message.Message.Chat.ID, session, "from")
		}
		b.answerCallback(ctx, botClient, callbackQuery.ID, "")

	case "et": // Edit To
		session.State = StateSelectingTo
		b.transitionState(session, StateSelectingTo)
		if callbackQuery.Message.Message != nil {
			b.showStationSelection(ctx, botClient, callbackQuery.Message.Message.Chat.ID, session, "to")
		}
		b.answerCallback(ctx, botClient, callbackQuery.ID, "")

	case "text_input": // Fallback to text input
		b.handleTextInputFallback(ctx, botClient, callbackQuery, session)

	case "noop": // No operation (pagination indicator)
		b.answerCallback(ctx, botClient, callbackQuery.ID, "")

	default:
		// Legacy support for old callback format
		if strings.HasPrefix(callbackQuery.Data, "train:") || callbackQuery.Data == "cancel" {
			b.handleLegacyCallback(ctx, botClient, callbackQuery, session)
		} else {
			b.answerCallback(ctx, botClient, callbackQuery.ID, "Неизвестная команда")
		}
	}
}

// handleCancel handles cancel action
func (b *Bot) handleCancel(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, session *UserSession) {
	b.clearSession(callbackQuery.From.ID)

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
			ParseMode: models.ParseModeMarkdown,
		})
		if err == nil {
			b.answerCallback(ctx, botClient, callbackQuery.ID, "Отменено")
			return
		}
	}

	if chatID == 0 {
		chatID = callbackQuery.From.ID
	}
	_, _ = botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "❌ *Создание поездки отменено*\n\nВы можете начать заново командой /newtrip",
		ParseMode: models.ParseModeMarkdown,
	})
	b.answerCallback(ctx, botClient, callbackQuery.ID, "Отменено")
}

// handleTrainSelect handles train selection and trip confirmation
func (b *Bot) handleTrainSelect(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, session *UserSession, params []string) {
	if len(params) == 0 {
		sendCallbackError(ctx, botClient, callbackQuery, "Ошибка: неверные параметры")
		return
	}

	index, err := strconv.Atoi(params[0])
	if err != nil {
		sendCallbackError(ctx, botClient, callbackQuery, "Ошибка формата")
		return
	}

	if session.Schedule == nil || index < 0 || index >= len(session.Schedule) {
		sendCallbackError(ctx, botClient, callbackQuery, "Ошибка: расписание не найдено")
		return
	}

	opt := session.Schedule[index]

	user, err := b.userUC.GetUserByTelegramID(ctx, callbackQuery.From.ID)
	if err != nil {
		sendCallbackError(ctx, botClient, callbackQuery, "Ошибка получения данных пользователя")
		return
	}

	tr := &domain.Trip{
		UserID:        user.ID,
		From:          session.From,
		To:            session.To,
		DepartureTime: opt.DepartureTime,
	}

	err = b.tripUC.ConfirmTrip(ctx, tr)
	if err != nil {
		sendCallbackError(ctx, botClient, callbackQuery, fmt.Sprintf("Ошибка: %s", err.Error()))
		return
	}

	b.clearSession(callbackQuery.From.ID)

	depTime := opt.DepartureTime.Format("02.01.2006 15:04")
	escapedTrainID := escapeMarkdown(opt.TrainID)
	escapedFrom := escapeMarkdown(session.FromName)
	escapedTo := escapeMarkdown(session.ToName)

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
			ParseMode: models.ParseModeMarkdown,
		})
		if err == nil {
			b.answerCallback(ctx, botClient, callbackQuery.ID, "Поездка создана!")
			return
		}
	}

	if chatID == 0 {
		chatID = callbackQuery.From.ID
	}
	_, _ = botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      successText,
		ParseMode: models.ParseModeMarkdown,
	})
	b.answerCallback(ctx, botClient, callbackQuery.ID, "Поездка создана!")
}

// handleTextInputFallback switches to text input mode
func (b *Bot) handleTextInputFallback(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, session *UserSession) {
	var text string
	var newState UserState

	if session.State == StateSelectingFrom {
		text = "⌨️ *Введите название или код станции отправления*\n\nНапример: Таганрог или s9613483"
		newState = StateWaitingFrom
	} else if session.State == StateSelectingTo {
		text = "⌨️ *Введите название или код станции назначения*\n\nНапример: Ростов-на-Дону или s9612913"
		newState = StateWaitingTo
	} else {
		b.answerCallback(ctx, botClient, callbackQuery.ID, "Ошибка состояния")
		return
	}

	session.State = newState
	b.transitionState(session, newState)

	var chatID int64
	if callbackQuery.Message.Message != nil {
		chatID = callbackQuery.Message.Message.Chat.ID
	} else {
		chatID = callbackQuery.From.ID
	}

	_, _ = botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	b.answerCallback(ctx, botClient, callbackQuery.ID, "")
}

// handleLegacyCallback handles old callback format for backward compatibility
func (b *Bot) handleLegacyCallback(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, session *UserSession) {
	if strings.HasPrefix(callbackQuery.Data, "train:") {
		params := []string{strings.TrimPrefix(callbackQuery.Data, "train:")}
		b.handleTrainSelect(ctx, botClient, callbackQuery, session, params)
	} else if callbackQuery.Data == "cancel" {
		b.handleCancel(ctx, botClient, callbackQuery, session)
	}
}

func sendCallbackError(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, message string) {
	_, _ = botClient.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackQuery.ID,
		Text:            message,
		ShowAlert:       true,
	})
}

// answerCallback is a helper to answer callback queries
func (b *Bot) answerCallback(ctx context.Context, botClient *bot.Bot, callbackID string, text string) {
	_, _ = botClient.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
}

// showStationSelection displays inline keyboard with recent and popular stations
func (b *Bot) showStationSelection(ctx context.Context, botClient *bot.Bot, chatID int64, session *UserSession, mode string) {
	var text string
	if mode == "from" {
		text = "📍 Выберите станцию отправления\n\nВыберите из недавних или популярных:"
	} else {
		text = "📍 Выберите станцию назначения\n\nВыберите из недавних или популярных:"
	}

	buttons := [][]models.InlineKeyboardButton{}

	// Recent stations (max 3)
	if len(session.RecentStations) > 0 {
		for i, station := range session.RecentStations {
			if i >= 3 {
				break
			}
			buttons = append(buttons, []models.InlineKeyboardButton{
				{
					Text:         "🕒 " + station.DisplayName,
					CallbackData: fmt.Sprintf("ss:r%d", i),
				},
			})
		}
	}

	// Popular stations (top 7)
	for i := 0; i < 7 && i < len(utils.PopularStations); i++ {
		station := utils.PopularStations[i]
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text:         "📍 " + station.DisplayName,
				CallbackData: fmt.Sprintf("ss:p%d", i),
			},
		})
	}

	// Text input fallback
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⌨️ Ввести название", CallbackData: "text_input"},
	})

	// Navigation
	navRow := []models.InlineKeyboardButton{}
	if mode == "to" {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "◀️ Назад",
			CallbackData: "b",
		})
	}
	navRow = append(navRow, models.InlineKeyboardButton{
		Text:         "❌ Отменить",
		CallbackData: "x",
	})
	buttons = append(buttons, navRow)

	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		// No ParseMode - avoid markdown escaping issues
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
	if err != nil {
		log.Printf("Error sending station selection: %v", err)
	}
}

// buildScheduleKeyboard builds paginated schedule keyboard
func (b *Bot) buildScheduleKeyboard(schedules []*domain.Schedule, page int) [][]models.InlineKeyboardButton {
	buttons := [][]models.InlineKeyboardButton{}

	pageSize := 5
	totalPages := (len(schedules) + pageSize - 1) / pageSize
	start := page * pageSize
	end := start + pageSize
	if end > len(schedules) {
		end = len(schedules)
	}

	// Train buttons for current page
	for i := start; i < end; i++ {
		sch := schedules[i]
		depTime := sch.DepartureTime.Format("15:04")
		arrTime := sch.ArrivalTime.Format("15:04")
		duration := humanDurationFromSeconds(int(sch.Duration))

		buttonText := fmt.Sprintf("🚆 %s | %s → %s (%s)",
			sch.TrainID, depTime, arrTime, duration)

		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text:         buttonText,
				CallbackData: fmt.Sprintf("tr:%d", i),
			},
		})
	}

	// Pagination row
	if totalPages > 1 {
		navRow := []models.InlineKeyboardButton{}

		if page > 0 {
			navRow = append(navRow, models.InlineKeyboardButton{
				Text:         "◀️",
				CallbackData: fmt.Sprintf("sp:%d", page-1),
			})
		}

		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%d/%d", page+1, totalPages),
			CallbackData: "noop",
		})

		if page < totalPages-1 {
			navRow = append(navRow, models.InlineKeyboardButton{
				Text:         "▶️",
				CallbackData: fmt.Sprintf("sp:%d", page+1),
			})
		}

		buttons = append(buttons, navRow)
	}

	// Actions row
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "◀️ Назад", CallbackData: "b"},
		{Text: "✏️ Изменить", CallbackData: "ef"},
		{Text: "❌ Отменить", CallbackData: "x"},
	})

	return buttons
}

// handleBack handles back navigation
func (b *Bot) handleBack(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, session *UserSession) {
	if len(session.StateHistory) < 2 {
		b.answerCallback(ctx, botClient, callbackQuery.ID, "Нет предыдущего шага")
		return
	}

	// Pop current state
	session.StateHistory = session.StateHistory[:len(session.StateHistory)-1]
	previousState := session.StateHistory[len(session.StateHistory)-1]
	session.State = previousState

	chatID := callbackQuery.Message.Message.Chat.ID

	// Render appropriate screen
	switch previousState {
	case StateSelectingFrom:
		b.showStationSelection(ctx, botClient, chatID, session, "from")
	case StateSelectingTo:
		b.showStationSelection(ctx, botClient, chatID, session, "to")
	case StateShowingSchedule:
		b.sendScheduleMessage(ctx, botClient, chatID, session)
	default:
		session.State = StateNone
		b.clearSession(callbackQuery.From.ID)
	}

	b.answerCallback(ctx, botClient, callbackQuery.ID, "")
}

// handleSelectStation handles station selection from inline keyboard
func (b *Bot) handleSelectStation(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, session *UserSession, params []string) {
	if len(params) == 0 {
		b.answerCallback(ctx, botClient, callbackQuery.ID, "Ошибка: неверные параметры")
		return
	}

	var selectedStation utils.StationOption
	var found bool

	indexStr := params[0]
	if strings.HasPrefix(indexStr, "r") {
		// Recent station
		idx, err := strconv.Atoi(indexStr[1:])
		if err != nil || idx < 0 || idx >= len(session.RecentStations) {
			b.answerCallback(ctx, botClient, callbackQuery.ID, "Станция не найдена")
			return
		}
		selectedStation = session.RecentStations[idx]
		found = true
	} else if strings.HasPrefix(indexStr, "p") {
		// Popular station
		idx, err := strconv.Atoi(indexStr[1:])
		if err != nil {
			b.answerCallback(ctx, botClient, callbackQuery.ID, "Ошибка формата")
			return
		}
		selectedStation, found = utils.GetStationByIndex(idx)
	}

	if !found {
		b.answerCallback(ctx, botClient, callbackQuery.ID, "Станция не найдена")
		return
	}

	// Add to recent stations
	b.addToRecentStations(session, selectedStation)

	chatID := callbackQuery.Message.Message.Chat.ID

	// Update session based on current state
	if session.State == StateSelectingFrom {
		session.From = selectedStation.Code
		session.FromName = selectedStation.DisplayName

		// Transition to selecting "To"
		b.transitionState(session, StateSelectingTo)
		b.showStationSelection(ctx, botClient, chatID, session, "to")
		b.answerCallback(ctx, botClient, callbackQuery.ID, "✓ "+selectedStation.DisplayName)

	} else if session.State == StateSelectingTo {
		session.To = selectedStation.Code
		session.ToName = selectedStation.DisplayName

		// Search for schedules
		b.answerCallback(ctx, botClient, callbackQuery.ID, "Поиск расписания...")

		filteredOptions, err := b.tripUC.Search(ctx, session.From, session.To, session.Date)
		if err != nil {
			b.sendRecoverableError(ctx, botClient, chatID,
				fmt.Sprintf("Ошибка поиска: %v", err),
				[]models.InlineKeyboardButton{
					{Text: "🔄 Попробовать снова", CallbackData: "ef"},
					{Text: "❌ Отменить", CallbackData: "x"},
				})
			return
		}

		if len(filteredOptions) == 0 {
			b.sendRecoverableError(ctx, botClient, chatID,
				"Рейсы не найдены для этого маршрута.",
				[]models.InlineKeyboardButton{
					{Text: "🔄 Другие станции", CallbackData: "ef"},
					{Text: "❌ Отменить", CallbackData: "x"},
				})
			return
		}

		session.Schedule = filteredOptions
		session.SchedulePage = 0
		b.transitionState(session, StateShowingSchedule)
		b.sendScheduleMessage(ctx, botClient, chatID, session)
	}
}

// handleSchedulePage handles schedule pagination
func (b *Bot) handleSchedulePage(ctx context.Context, botClient *bot.Bot, callbackQuery *models.CallbackQuery, session *UserSession, params []string) {
	if len(params) == 0 {
		b.answerCallback(ctx, botClient, callbackQuery.ID, "Ошибка: неверная страница")
		return
	}

	page, err := strconv.Atoi(params[0])
	if err != nil {
		b.answerCallback(ctx, botClient, callbackQuery.ID, "Ошибка формата")
		return
	}

	session.SchedulePage = page

	// Update message with new page
	chatID := callbackQuery.Message.Message.Chat.ID
	messageID := callbackQuery.Message.Message.ID

	text := buildScheduleText(session.Schedule, session.FromName, session.ToName)
	keyboard := b.buildScheduleKeyboard(session.Schedule, page)

	_, err = botClient.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		// No ParseMode - avoid markdown escaping issues
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})

	if err != nil {
		log.Printf("Error editing message: %v", err)
		// Fallback: send new message
		b.sendScheduleMessage(ctx, botClient, chatID, session)
	}

	b.answerCallback(ctx, botClient, callbackQuery.ID, "")
}

// sendScheduleMessage sends schedule message with pagination
func (b *Bot) sendScheduleMessage(ctx context.Context, botClient *bot.Bot, chatID int64, session *UserSession) {
	text := buildScheduleText(session.Schedule, session.FromName, session.ToName)
	keyboard := b.buildScheduleKeyboard(session.Schedule, session.SchedulePage)

	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		// No ParseMode - avoid markdown escaping issues
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	if err != nil {
		log.Printf("Error sending schedule: %v", err)
	}
}

// sendRecoverableError sends error message with recovery action buttons
func (b *Bot) sendRecoverableError(ctx context.Context, botClient *bot.Bot, chatID int64, errorMsg string, actions []models.InlineKeyboardButton) {
	text := fmt.Sprintf("⚠️ Ошибка\n\n%s\n\nЧто делать?", errorMsg)

	buttons := [][]models.InlineKeyboardButton{}

	// Add action buttons in pairs
	for i := 0; i < len(actions); i += 2 {
		row := []models.InlineKeyboardButton{actions[i]}
		if i+1 < len(actions) {
			row = append(row, actions[i+1])
		}
		buttons = append(buttons, row)
	}

	_, err := botClient.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		// No ParseMode - avoid markdown escaping issues
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
	if err != nil {
		log.Printf("Error sending recoverable error: %v", err)
	}
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
	b.WriteString("🚆 Расписание рейсов\n\n")
	fmt.Fprintf(&b, "📍 %s → %s\n\n", from, to)
	b.WriteString("Выберите поезд:\n\n")

	for i, opt := range options {
		num := i + 1
		title := cleanTitle(opt.Title)

		dep := opt.DepartureTime.Format("02.01.2006 15:04")
		arr := opt.ArrivalTime.Format("15:04")
		durationStr := humanDurationFromSeconds(int(opt.Duration))

		fmt.Fprintf(&b, "%d. %s\n", num, title)
		fmt.Fprintf(&b, "   🚆 Поезд: %s\n", opt.TrainID)
		fmt.Fprintf(&b, "   🕒 %s → %s\n", dep, arr)
		fmt.Fprintf(&b, "   ⏱ %s\n\n", durationStr)
	}

	return b.String()
}