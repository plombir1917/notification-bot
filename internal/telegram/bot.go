// Package telegram содержит Telegram-бота: команды подписки и рассылку
// напоминаний подписчикам.
package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"notification-bot/internal/calendar"
	"notification-bot/internal/schedule"
	"notification-bot/internal/store"
)

// Тексты напоминаний.
const (
	// MsgNormal — обычное напоминание о подписании табеля.
	MsgNormal = "📋 Напоминание: не забудьте подписать табель учёта рабочего времени."

	// MsgDecember — спец-сообщение (16 декабря) про финальный табель.
	MsgDecember = "📋 Подписание финального табеля за декабрь отслеживайте " +
		"самостоятельно на корпоративной почте."
)

const (
	msgStart = "Привет! Вы подписались на напоминания о подписании табеля.\n" +
		"Напоминания приходят 15-го и 30-го числа (с учётом выходных и праздников) в 11:00 по МСК.\n\n" +
		"Команды:\n/status — когда следующее напоминание\n/stop — отписаться"
	msgAlreadyStarted = "Вы уже подписаны на напоминания. /status — когда следующее, /stop — отписаться."
	msgStop           = "Вы отписались от напоминаний. Чтобы снова подписаться — отправьте /start."
	msgNotSubscribed  = "Вы и так не подписаны. Отправьте /start, чтобы подписаться."
)

// Bot — обёртка над Telegram-ботом.
type Bot struct {
	b     *bot.Bot
	store *store.Store
	cal   *calendar.Calendar
	loc   *time.Location
}

// New создаёт бота и регистрирует обработчики команд.
func New(token string, st *store.Store, cal *calendar.Calendar, loc *time.Location) (*Bot, error) {
	tb := &Bot{store: st, cal: cal, loc: loc}

	opts := []bot.Option{
		bot.WithDefaultHandler(tb.handleDefault),
	}
	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("инициализация Telegram-бота: %w", err)
	}
	tb.b = b

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, tb.handleStart)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stop", bot.MatchTypePrefix, tb.handleStop)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/status", bot.MatchTypePrefix, tb.handleStatus)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/test", bot.MatchTypePrefix, tb.handleTest)

	return tb, nil
}

// Start запускает long polling; блокирует до отмены ctx.
func (tb *Bot) Start(ctx context.Context) {
	tb.b.Start(ctx)
}

func (tb *Bot) handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	added, err := tb.store.AddSubscriber(chatID)
	if err != nil {
		log.Printf("не удалось добавить подписчика %d: %v", chatID, err)
		tb.reply(ctx, chatID, "Не удалось подписать вас, попробуйте позже.")
		return
	}
	if added {
		log.Printf("новый подписчик: %d", chatID)
		tb.reply(ctx, chatID, msgStart)
	} else {
		tb.reply(ctx, chatID, msgAlreadyStarted)
	}
}

func (tb *Bot) handleStop(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	removed, err := tb.store.RemoveSubscriber(chatID)
	if err != nil {
		log.Printf("не удалось удалить подписчика %d: %v", chatID, err)
		tb.reply(ctx, chatID, "Не удалось отписать вас, попробуйте позже.")
		return
	}
	if removed {
		log.Printf("подписчик отписался: %d", chatID)
		tb.reply(ctx, chatID, msgStop)
	} else {
		tb.reply(ctx, chatID, msgNotSubscribed)
	}
}

func (tb *Bot) handleStatus(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	now := time.Now().In(tb.loc)
	date, action := schedule.NextReminder(now, tb.cal)
	tb.reply(ctx, update.Message.Chat.ID, statusText(date, action))
}

// handleTest присылает вызвавшему пример напоминания — для ручной проверки
// отправки, не дожидаясь запланированной даты.
func (tb *Bot) handleTest(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	log.Printf("тестовое напоминание по запросу %d", chatID)
	tb.reply(ctx, chatID, "🧪 Тестовое сообщение. Так выглядит напоминание:")
	tb.reply(ctx, chatID, MsgNormal)
}

func (tb *Bot) handleDefault(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	tb.reply(ctx, update.Message.Chat.ID,
		"Команды: /start — подписаться, /stop — отписаться, /status — когда следующее напоминание.")
}

// Broadcast отправляет текст всем подписчикам. Подписчиков, заблокировавших
// бота (403), удаляет из хранилища.
func (tb *Bot) Broadcast(ctx context.Context, text string) {
	ids := tb.store.Subscribers()
	log.Printf("рассылка напоминания %d подписчикам", len(ids))
	for _, id := range ids {
		_, err := tb.b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: id,
			Text:   text,
		})
		if err == nil {
			continue
		}
		if isUnreachable(err) {
			log.Printf("подписчик %d недоступен (%v), удаляю", id, err)
			if _, rerr := tb.store.RemoveSubscriber(id); rerr != nil {
				log.Printf("не удалось удалить подписчика %d: %v", id, rerr)
			}
		} else {
			log.Printf("ошибка отправки подписчику %d: %v", id, err)
		}
	}
}

func (tb *Bot) reply(ctx context.Context, chatID int64, text string) {
	if _, err := tb.b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		log.Printf("ошибка ответа %d: %v", chatID, err)
	}
}

// isUnreachable распознаёт ошибки, означающие, что подписчику больше нельзя писать.
func isUnreachable(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "bot was blocked") ||
		strings.Contains(s, "user is deactivated") ||
		strings.Contains(s, "chat not found") ||
		strings.Contains(s, "forbidden")
}

func statusText(date time.Time, action schedule.Action) string {
	if action == schedule.None {
		return "Ближайшее напоминание не запланировано."
	}
	when := formatRuDate(date)
	if action == schedule.SendDecemberNotice {
		return fmt.Sprintf("Ближайшее сообщение: %s — про финальный декабрьский табель.", when)
	}
	return fmt.Sprintf("Ближайшее напоминание о подписании табеля: %s в 11:00 по МСК.", when)
}

var ruMonths = [...]string{
	"", "января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}

func formatRuDate(d time.Time) string {
	return fmt.Sprintf("%d %s %d", d.Day(), ruMonths[int(d.Month())], d.Year())
}
