package bot

import (
	"log"
	"sync"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Lang string

const (
	LangEN Lang = "en"
	LangRU Lang = "ru"
)

var langPrefs = struct {
	mu    sync.RWMutex
	byUID map[int64]Lang
}{
	byUID: make(map[int64]Lang),
}

func setUserLang(userID int64, l Lang) {
	langPrefs.mu.Lock()
	langPrefs.byUID[userID] = l
	langPrefs.mu.Unlock()
}

func getUserLang(userID int64) Lang {
	langPrefs.mu.RLock()
	l, ok := langPrefs.byUID[userID]
	langPrefs.mu.RUnlock()
	if ok {
		return l
	}
	// default to EN if not set
	return LangEN
}

// i18n dictionaries
var i18n = map[Lang]map[string]string{
	LangEN: {
		"settings.title":                     "⚙️ Settings",
		"settings.current_mode":              "• Download mode: *%s*",
		"settings.current_lang":              "• Language: %s",
		"settings.choose_mode":               "🎛 Choose download mode",
		"settings.back":                      "⬅️ Back",
		"settings.mode.caption":              "Download mode:",
		"settings.mode.caption.friendly":     "Pick how to save albums:",
		"settings.mode.saved":                "Saved mode: %s",
		"settings.mode.saved.friendly":       "✅ Mode saved: %s",
		"settings.unknown_action":            "Unknown action.",
		"settings.language":                  "🌍 Choose language",
		"settings.language.caption":          "Language:",
		"settings.language.caption.friendly": "Pick your language:",
		"settings.language.saved":            "Language saved: %s",
		"settings.language.saved.friendly":   "✅ Language set to: %s",
		"mode.media":                         "media",
		"mode.files":                         "files",
		"mode.ask":                           "ask",
		"settings.quality":                   "Media quality",
		"settings.quality.caption":           "Media quality:",
		"settings.quality.saved":             "Saved quality: %s",
		"quality.ask":                        "ask",
		"quality.original":                   "original",
		"quality.high":                       "high",
		"quality.low":                        "low",
		"album.ask":                          "Send album as media or files?",
		"album.button.media":                 "Media",
		"album.button.file":                  "Files",
		"msg.request_post":                   "Drop a Reddit link here — I’ll fetch it ✨",
		"msg.no_media_found":                 "Hmm, no media found in that post.",
		"msg.select_quality":                 "Choose the quality:",
		"err.panic":                          "Something went wrong (panic).",
		"err.broken_callback":                "Broken callback data.",
		"err.resend_link":                    "Please resend the link.",
		"err.internal":                       "Internal error.",
		"unknown.type":                       "Unknown type (please report it on GitHub).",
		"cmd.start":                          "Welcome! This bot downloads media from Reddit posts — just send me a link, for example:\nhttps://www.reddit.com/r/TheCatternet/comments/1nrw9xt/she_grow_up/\n\nCommands:\n/start — start\n/settings — settings\n/help — help",
		"cmd.help":                           "Send a Reddit link. Text becomes text; images/videos get uploaded with title & link.",
		"cmd.desc.start":                     "Start the bot",
		"cmd.desc.help":                      "How to use the bot",
		"cmd.desc.settings":                  "Open settings",
	},
	LangRU: {
		"settings.title":                     "⚙️ Настройки",
		"settings.current_mode":              "• Режим скачивания: *%s*",
		"settings.current_lang":              "• Язык: %s",
		"settings.choose_mode":               "🎛 Выбрать режим скачивания",
		"settings.back":                      "⬅️ Назад",
		"settings.mode.caption":              "Режим скачивания:",
		"settings.mode.caption.friendly":     "Как сохранять альбомы?",
		"settings.mode.saved":                "Режим сохранён: %s",
		"settings.mode.saved.friendly":       "✅ Режим сохранён: %s",
		"settings.unknown_action":            "Неизвестное действие.",
		"settings.language":                  "🌍 Выбрать язык",
		"settings.language.caption":          "Язык:",
		"settings.language.caption.friendly": "Выберите язык:",
		"settings.language.saved":            "Язык сохранён: %s",
		"settings.language.saved.friendly":   "✅ Язык установлен: %s",
		"mode.media":                         "Медиа",
		"mode.files":                         "Файлы",
		"mode.ask":                           "Спрашивать",
		"settings.quality":                   "Качество медиа",
		"settings.quality.caption":           "Качество медиа:",
		"settings.quality.saved":             "Качество сохранено: %s",
		"quality.ask":                        "спрашивать",
		"quality.original":                   "оригинальное",
		"quality.high":                       "высокое",
		"quality.low":                        "низкое",

		"settings.link":         "Прикреплять ссылку",
		"settings.link.caption": "Ссылка к посту:",
		"settings.link.saved":   "Настройка сохранена: %s",
		"link.on":               "да",
		"link.off":              "нет",
		"album.ask":             "Отправить альбом как медиа или файлами?",
		"album.button.media":    "Медиа",
		"album.button.file":     "Файлы",
		"msg.request_post":      "Кидай ссылку на Reddit — всё принесу ✨",
		"msg.no_media_found":    "Похоже, в посте нет медиа.",
		"msg.select_quality":    "Выберите качество:",
		"err.panic":             "Что-то пошло не так (panic).",
		"err.broken_callback":   "Некорректные callback-данные.",
		"err.resend_link":       "Пришлите ссылку ещё раз.",
		"err.internal":          "Внутренняя ошибка.",
		"unknown.type":          "Неизвестный тип (сообщите в репозитории).",
		"cmd.start":             "Добро пожаловать! Бот умеет скачивать медиа из постов Reddit — просто пришли мне ссылку, например:\nhttps://www.reddit.com/r/TheCatternet/comments/1nrw9xt/she_grow_up/\n\nКоманды:\n/start — старт\n/settings — настройки\n/help — помощь",
		"cmd.help":              "Пришлите ссылку на Reddit. Текст — текстом, картинки/видео — загружу с заголовком и ссылкой.",
		"cmd.desc.start":        "Запустить бота",
		"cmd.desc.help":         "Как пользоваться ботом",
		"cmd.desc.settings":     "Открыть настройки",
	},
}

func tr(l Lang, key string) string {
	if m, ok := i18n[l]; ok {
		if s, ok2 := m[key]; ok2 {
			return s
		}
	}
	// fallback to EN
	v := i18n[LangEN][key]
	if v == "" {
		return key
	}
	return v
}

func t(uid int64, key string) string {
	return tr(getUserLang(uid), key)
}

// ----- Bot commands (per-language) -----
func commandsFor(lang Lang) []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{Command: "start", Description: tr(lang, "cmd.desc.start")},
		{Command: "settings", Description: tr(lang, "cmd.desc.settings")},
		{Command: "help", Description: tr(lang, "cmd.desc.help")},
	}
}

func installCommands(bot *gotgbot.Bot) {
	if _, err := bot.SetMyCommands(commandsFor(LangEN), &gotgbot.SetMyCommandsOpts{
		Scope:        gotgbot.BotCommandScopeDefault{},
		LanguageCode: "",
	}); err != nil {
		log.Println("SetMyCommands (default) failed:", err)
	}
	if _, err := bot.SetMyCommands(commandsFor(LangEN), &gotgbot.SetMyCommandsOpts{
		Scope:        gotgbot.BotCommandScopeDefault{},
		LanguageCode: "en",
	}); err != nil {
		log.Println("SetMyCommands (en) failed:", err)
	}
	if _, err := bot.SetMyCommands(commandsFor(LangRU), &gotgbot.SetMyCommandsOpts{
		Scope:        gotgbot.BotCommandScopeDefault{},
		LanguageCode: "ru",
	}); err != nil {
		log.Println("SetMyCommands (ru) failed:", err)
	}
}
