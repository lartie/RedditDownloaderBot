package bot

import "sync"

// ----- Localization (in-memory) -----

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
		"settings.title":            "Settings",
		"settings.current_mode":     "Current download mode: *%s*",
		"settings.choose_mode":      "Choose download mode",
		"settings.back":             "⬅️ Back",
		"settings.mode.caption":     "Download mode:",
		"settings.mode.saved":       "Saved mode: %s",
		"settings.unknown_action":   "Unknown settings action.",
		"settings.language":         "Choose language",
		"settings.language.caption": "Language:",
		"settings.language.saved":   "Language saved: %s",
		"mode.media":                "media",
		"mode.files":                "files",
		"mode.ask":                  "ask",
		"album.ask":                 "Download album as media or file?",
		"album.button.media":        "Media",
		"album.button.file":         "File",
		"msg.request_post":          "Please send a Reddit post.",
		"msg.no_media_found":        "No media found.",
		"msg.select_quality":        "Please select the quality.",
		"err.panic":                 "Cannot get data. (panic)",
		"err.broken_callback":       "Broken callback data",
		"err.resend_link":           "Please resend the link.",
		"err.internal":              "Internal error",
		"unknown.type":              "Unknown type (Please report this on the main GitHub project.)",
		"cmd.start":                 "Hey!\n\nJust send me a post or comment, and I’ll download it for you.",
		"cmd.about":                 "Reddit Downloader Bot v%v\nBy Hirbod Behnam\nSource: https://github.com/HirbodBehnam/RedditDownloaderBot",
		"cmd.help":                  "You can send me Reddit posts or comments. If it’s text only, I’ll send a text message. If it’s an image or video, I’ll upload and send the content along with the title and link.",
	},
	LangRU: {
		"settings.title":            "Настройки",
		"settings.current_mode":     "Текущий режим скачивания: *%s*",
		"settings.choose_mode":      "Выбрать режим скачивания",
		"settings.back":             "⬅️ Назад",
		"settings.mode.caption":     "Режим скачивания:",
		"settings.mode.saved":       "Режим сохранён: %s",
		"settings.unknown_action":   "Неизвестное действие настроек.",
		"settings.language":         "Выбрать язык",
		"settings.language.caption": "Язык:",
		"settings.language.saved":   "Язык сохранён: %s",
		"mode.media":                "медиа",
		"mode.files":                "файлы",
		"mode.ask":                  "спрашивать",
		"album.ask":                 "Скачать альбом как медиа или файлы?",
		"album.button.media":        "Медиа",
		"album.button.file":         "Файлы",
		"msg.request_post":          "Пришлите ссылку на пост в Reddit.",
		"msg.no_media_found":        "Медиа не найдено.",
		"msg.select_quality":        "Выберите качество.",
		"err.panic":                 "Не удалось получить данные (panic).",
		"err.broken_callback":       "Некорректные данные callback.",
		"err.resend_link":           "Пожалуйста, пришлите ссылку заново.",
		"err.internal":              "Внутренняя ошибка.",
		"unknown.type":              "Неизвестный тип (отправьте отчёт в репозиторий).",
		"cmd.start":                 "Привет!\n\nПросто пришли пост или комментарий — я его скачаю.",
		"cmd.about":                 "Reddit Downloader Bot v%v\nАвтор: Hirbod Behnam\nSource: https://github.com/HirbodBehnam/RedditDownloaderBot",
		"cmd.help":                  "Можешь присылать посты и комментарии Reddit. Текст пришлю текстом, изображения/видео — загружу с заголовком и ссылкой.",
	},
}

func tr(l Lang, key string) string {
	if m, ok := i18n[l]; ok {
		if s, ok2 := m[key]; ok2 {
			return s
		}
	}
	// fallback to EN
	return i18n[LangEN][key]
}

func t(uid int64, key string) string {
	return tr(getUserLang(uid), key)
}
