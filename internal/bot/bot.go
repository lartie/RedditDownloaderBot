package bot

import (
	"RedditDownloaderBot/internal/cache"
	"RedditDownloaderBot/pkg/common"
	"RedditDownloaderBot/pkg/reddit"
	"RedditDownloaderBot/pkg/util"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"

	"github.com/google/uuid"
)

// ----- Settings (in-memory) -----

type DownloadMode int

const (
	DownloadModeAsk DownloadMode = iota
	DownloadModeMedia
	DownloadModeFiles
)

const (
	KindSettings = "s"

	ActionSetLang  = "sl"
	ActionOpenLang = "ol"
	ActionOpenMode = "om"
	ActionSetMode  = "sm"
	ActionOpenRoot = "or"
)

func (m DownloadMode) String() string {
	switch m {
	case DownloadModeMedia:
		return "media"
	case DownloadModeFiles:
		return "files"
	default:
		return "ask"
	}
}

func parseDownloadMode(s string) DownloadMode {
	switch strings.ToLower(s) {
	case "media":
		return DownloadModeMedia
	case "files":
		return DownloadModeFiles
	default:
		return DownloadModeAsk
	}
}

// user preferences stored in RAM
var userPrefs = struct {
	mu    sync.RWMutex
	byUID map[int64]DownloadMode
}{
	byUID: make(map[int64]DownloadMode),
}

func getUserMode(userID int64) DownloadMode {
	userPrefs.mu.RLock()
	m, ok := userPrefs.byUID[userID]
	userPrefs.mu.RUnlock()
	if !ok {
		return DownloadModeAsk
	}
	return m
}

func setUserMode(userID int64, m DownloadMode) {
	userPrefs.mu.Lock()
	userPrefs.byUID[userID] = m
	userPrefs.mu.Unlock()
}

// UI builders
func settingsRootKeyboardFor(uid int64) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{
					Text:         t(uid, "settings.choose_mode"),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionOpenMode, "").String(),
				},
			},
			{
				{
					Text:         t(uid, "settings.language"),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionOpenLang, "").String(),
				},
			},
			{
				{
					Text:         t(uid, "settings.back"),
					CallbackData: NewSettingsCallbackData(KindSettings, "back", "").String(),
				},
			},
		},
	}
}

func settingsModeKeyboard(uid int64, current DownloadMode) gotgbot.InlineKeyboardMarkup {
	mark := func(label string, active bool) string {
		if active {
			return "• " + label + " ✅"
		}
		return label
	}
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{
					Text:         mark(t(uid, "mode.media"), current == DownloadModeMedia),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionSetMode, "media").String(),
				},
				{
					Text:         mark(t(uid, "mode.files"), current == DownloadModeFiles),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionSetMode, "files").String(),
				},
			},
			{
				{
					Text:         mark(t(uid, "mode.ask"), current == DownloadModeAsk),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionSetMode, "ask").String(),
				},
			},
			{
				{
					Text:         t(uid, "settings.back"),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionOpenRoot, "").String(),
				},
			},
		},
	}
}

func settingsLangKeyboard(uid int64) gotgbot.InlineKeyboardMarkup {
	cur := getUserLang(uid)
	mark := func(code string, label string) string {
		if string(cur) == code {
			return "• " + label + " ✅"
		}
		return label
	}
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{
					Text:         mark("en", "English"),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionSetLang, "en").String(),
				},
				{
					Text:         mark("ru", "Русский"),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionSetLang, "ru").String(),
				},
			},
			{
				{
					Text:         t(uid, "settings.back"),
					CallbackData: NewSettingsCallbackData(KindSettings, ActionSetLang, "").String(),
				},
			},
		},
	}
}

// RunBot runs the bot with the specified token
func (c *Client) RunBot(token string, allowedUsers AllowedUsers) {
	// Setup the bot
	bot, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		BotClient: gotgbot.BotClient(&gotgbot.BaseBotClient{
			DefaultRequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 20,
			},
		}),
	})
	if err != nil {
		log.Fatal("Cannot initialize the bot:", err.Error())
	}
	log.Println("Bot authorized on account.", bot.Username)
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			log.Println("An error occurred while handling update: ", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})
	updater := ext.NewUpdater(dispatcher, nil)
	// Add handlers
	dispatcher.AddHandler(handlers.NewCallback(func(_ *gotgbot.CallbackQuery) bool {
		return true
	}, c.handleCallback))
	dispatcher.AddHandler(handlers.NewMessage(func(msg *gotgbot.Message) bool {
		return allowedUsers.IsAllowed(msg.From.Id)
	}, c.handleMessage))
	// Wait for updates
	err = updater.StartPolling(bot, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 60,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 60,
			},
		},
	})
	if err != nil {
		panic("Failed to start polling: " + err.Error())
	}
	log.Printf("%s has been started . . .\n", bot.User.Username)

	// Idle, to keep updates coming in, and avoid bot stopping.
	updater.Idle()
}

func (c *Client) handleMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	// Only text messages are allowed
	if ctx.Message.Text == "" {
		_, err := ctx.EffectiveChat.SendMessage(bot, t(ctx.Message.From.Id, "msg.request_post"), nil)
		return err
	}
	// Check if the message is command. I don't use command handler because I'll lose
	// the userID control.
	switch ctx.Message.Text {
	case "/start":
		_, err := ctx.EffectiveChat.SendMessage(bot, tr(getUserLang(ctx.Message.From.Id), "cmd.start"), nil)
		return err
	case "/about":
		_, err := ctx.EffectiveChat.SendMessage(bot, fmt.Sprintf(tr(getUserLang(ctx.Message.From.Id), "cmd.about"), common.Version), nil)
		return err
	case "/help":
		_, err := ctx.EffectiveChat.SendMessage(bot, tr(getUserLang(ctx.Message.From.Id), "cmd.help"), nil)
		return err
	case "/settings":
		uid := ctx.Message.From.Id
		mode := getUserMode(uid)
		title := tr(getUserLang(uid), "settings.title")
		current := tr(getUserLang(uid), "settings.current_mode")
		text := title + "\n\n" + fmt.Sprintf(current, mode.String())
		_, err := ctx.EffectiveChat.SendMessage(bot, text, &gotgbot.SendMessageOpts{
			ParseMode:   gotgbot.ParseModeMarkdownV2,
			ReplyMarkup: settingsRootKeyboardFor(uid),
		})
		return err
	default:
		return c.fetchPostDetailsAndSend(bot, ctx)
	}
}

// fetchPostDetailsAndSend gets the basic info about the post being sent to us
func (c *Client) fetchPostDetailsAndSend(bot *gotgbot.Bot, ctx *ext.Context) error {
	result, realPostUrl, fetchErr := c.RedditOauth.StartFetch(ctx.Message.Text)
	if fetchErr != nil {
		if fetchErr.NormalError != "" {
			log.Println("Cannot fetch the post", ctx.Message.Text, ":", fetchErr.NormalError)
		}
		_, err := ctx.EffectiveMessage.Reply(bot, fetchErr.BotError, nil)
		return err
	}
	// Check the result type
	toSendText := ""
	toSendOpt := &gotgbot.SendMessageOpts{
		ParseMode: gotgbot.ParseModeMarkdownV2,
	}
	switch data := result.(type) {
	case reddit.FetchResultText:
		toSendText = addLinkIfNeeded(data.Title+"\n"+data.Text, realPostUrl)
	case reddit.FetchResultComment:
		toSendText = addLinkIfNeeded(data.Text, realPostUrl)
	case reddit.FetchResultMedia:
		if len(data.Medias) == 0 {
			toSendText = t(ctx.Message.From.Id, "msg.no_media_found")
			break
		}
		// If there is one media quality, download it
		// Also allow the user to choose between photo or document in image
		if len(data.Medias) == 1 && data.Type != reddit.FetchResultMediaTypePhoto {
			switch data.Type {
			case reddit.FetchResultMediaTypeGif:
				return c.handleGifUpload(bot, data.Medias[0].Link, data.Title, data.ThumbnailLinks.SelectThumbnail(maxThumbnailDimensions), realPostUrl, data.Description, data.Medias[0].Dim, ctx.EffectiveChat.Id)
			case reddit.FetchResultMediaTypeVideo:
				// If the video does have an audio, ask user if they want the audio
				if _, hasAudio := data.HasAudio(); !hasAudio {
					// Otherwise, just download the video
					return c.handleVideoUpload(bot, data.Medias[0].Link, "", data.Title, data.ThumbnailLinks.SelectThumbnail(maxThumbnailDimensions), realPostUrl, data.Description, data.Medias[0].Dim, data.Duration, ctx.EffectiveChat.Id)
				}
			default:
				panic("Shash")
			}
		}
		// Allow the user to select quality
		toSendText = t(ctx.Message.From.Id, "msg.select_quality")
		idString := util.UUIDToBase64(uuid.New())
		audioIndex, _ := data.HasAudio()
		switch data.Type {
		case reddit.FetchResultMediaTypePhoto:
			toSendOpt.ReplyMarkup = createPhotoInlineKeyboard(idString, data)
		case reddit.FetchResultMediaTypeGif:
			toSendOpt.ReplyMarkup = createGifInlineKeyboard(idString, data)
		case reddit.FetchResultMediaTypeVideo:
			toSendOpt.ReplyMarkup = createVideoInlineKeyboard(idString, data)
		}
		// Insert the id in cache
		err := c.CallbackCache.SetMediaCache(idString, cache.CallbackDataCached{
			PostLink:      realPostUrl,
			Links:         getLinkMapOfFetchResultMediaEntries(data.Medias),
			Title:         data.Title,
			ThumbnailLink: data.ThumbnailLinks.SelectThumbnail(maxThumbnailDimensions),
			Description:   data.Description,
			Type:          data.Type,
			Duration:      data.Duration,
			AudioIndex:    audioIndex,
		})
		if err != nil {
			log.Println("Cannot set the media cache in database:", err)
		}
	case reddit.FetchResultAlbum:
		// auto-apply user preference if not "ask"
		uid := ctx.Message.From.Id
		switch getUserMode(uid) {
		case DownloadModeMedia:
			return c.handleAlbumUpload(bot, data, realPostUrl, ctx.EffectiveChat.Id, false)
		case DownloadModeFiles:
			return c.handleAlbumUpload(bot, data, realPostUrl, ctx.EffectiveChat.Id, true)
		}
		idString := util.UUIDToBase64(uuid.New())
		err := c.CallbackCache.SetAlbumCache(idString, cache.CallbackAlbumCached{
			PostLink: realPostUrl,
			Album:    data,
		})
		if err != nil {
			log.Println("Cannot set the album cache in database:", err)
		}
		toSendText = t(ctx.Message.From.Id, "album.ask")
		uid = ctx.Message.From.Id
		toSendOpt.ReplyMarkup = gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{
				gotgbot.InlineKeyboardButton{
					Text: t(uid, "album.button.media"),
					CallbackData: CallbackButtonData{
						ID:   idString,
						Mode: CallbackButtonDataModePhoto,
					}.String(),
				},
				gotgbot.InlineKeyboardButton{
					Text: t(uid, "album.button.file"),
					CallbackData: CallbackButtonData{
						ID:   idString,
						Mode: CallbackButtonDataModeFile,
					}.String(),
				},
			}},
		}
	default:
		log.Printf("unknown type: %T\n", result)
		toSendText = tr(getUserLang(ctx.Message.From.Id), "unknown.type")
	}
	// Check the toSendText size
	if len(toSendText) > 4096 {
		_, err := bot.SendDocument(ctx.EffectiveChat.Id, &gotgbot.FileReader{
			Name: "post.txt",
			Data: strings.NewReader(toSendText),
		}, &gotgbot.SendDocumentOpts{ReplyParameters: &gotgbot.ReplyParameters{
			MessageId: ctx.EffectiveMessage.MessageId,
		}})
		return err
	}
	_, err := ctx.EffectiveMessage.Reply(bot, toSendText, toSendOpt)
	if err != nil {
		toSendOpt.ParseMode = gotgbot.ParseModeMarkdown // fall to V1 and try again
		_, err = ctx.EffectiveMessage.Reply(bot, toSendText, toSendOpt)
		if err != nil {
			toSendOpt.ParseMode = gotgbot.ParseModeNone // fall back again and don't format message
			_, err = ctx.EffectiveMessage.Reply(bot, toSendText, toSendOpt)
		}
	}
	return err
}

// handleCallback handles the callback query of selecting a quality for any media type
func (c *Client) handleCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	// Don't crash!
	defer func() {
		if r := recover(); r != nil {
			_, _ = ctx.EffectiveChat.SendMessage(bot, t(ctx.CallbackQuery.From.Id, "err.panic"), nil)
			log.Println("Recovering from panic:", r)
		}
	}()
	// Delete the message
	_, _ = bot.DeleteMessage(ctx.EffectiveChat.Id, ctx.EffectiveMessage.GetMessageId(), nil)
	// Settings callbacks (identified by kind == "settings")
	var scd settingsCallbackData
	if err := json.Unmarshal([]byte(ctx.CallbackQuery.Data), &scd); err == nil && scd.Kind == KindSettings {
		uid := ctx.CallbackQuery.From.Id
		switch scd.Action {
		case ActionOpenMode:
			_, err := ctx.EffectiveChat.SendMessage(bot, t(uid, "settings.mode.caption"), &gotgbot.SendMessageOpts{
				ReplyMarkup: settingsModeKeyboard(uid, getUserMode(uid)),
			})
			return err
		case ActionOpenRoot, "back":
			text := tr(getUserLang(uid), "settings.title")
			_, err := ctx.EffectiveChat.SendMessage(bot, text, &gotgbot.SendMessageOpts{
				ReplyMarkup: settingsRootKeyboardFor(uid),
			})
			return err
		case ActionSetMode:
			m := parseDownloadMode(scd.Value)
			setUserMode(uid, m)
			_, err := ctx.EffectiveChat.SendMessage(bot, fmt.Sprintf(tr(getUserLang(uid), "settings.mode.saved"), parseDownloadMode(scd.Value).String()), &gotgbot.SendMessageOpts{
				ReplyMarkup: settingsModeKeyboard(uid, m),
			})
			return err
		case ActionOpenLang:
			_, err := ctx.EffectiveChat.SendMessage(bot, t(uid, "settings.language.caption"), &gotgbot.SendMessageOpts{
				ReplyMarkup: settingsLangKeyboard(uid),
			})
			return err
		case ActionSetLang:
			switch strings.ToLower(scd.Value) {
			case "ru":
				setUserLang(uid, LangRU)
			default:
				setUserLang(uid, LangEN)
			}
			cur := getUserLang(uid)
			_, err := ctx.EffectiveChat.SendMessage(bot, fmt.Sprintf(tr(cur, "settings.language.saved"), strings.ToUpper(string(cur))), &gotgbot.SendMessageOpts{
				ReplyMarkup: settingsLangKeyboard(uid),
			})
			return err
		default:
			_, err := ctx.EffectiveChat.SendMessage(bot, t(uid, "settings.unknown_action"), nil)
			return err
		}
	}
	// Parse the data
	var data CallbackButtonData
	err := json.Unmarshal([]byte(ctx.CallbackQuery.Data), &data)
	if err != nil {
		uid := ctx.CallbackQuery.From.Id
		_, err = ctx.EffectiveChat.SendMessage(bot, t(uid, "err.broken_callback"), nil)
		return err
	}
	// Get the cache from database
	cachedData, err := c.CallbackCache.GetAndDeleteMediaCache(data.ID)
	if errors.Is(err, cache.NotFoundErr) {
		// Check albums
		var album cache.CallbackAlbumCached
		album, err = c.CallbackCache.GetAndDeleteAlbumCache(data.ID)
		if err == nil {
			return c.handleAlbumUpload(bot, album.Album, album.PostLink, ctx.EffectiveChat.Id, data.Mode == CallbackButtonDataModeFile)
		} else if errors.Is(err, cache.NotFoundErr) {
			// It does not exist...
			uid := ctx.CallbackQuery.From.Id
			_, err = ctx.EffectiveChat.SendMessage(bot, t(uid, "err.resend_link"), nil)
			return err
		}
		// Fall to report internal error
	}
	// Check other errors
	if err != nil {
		uid := ctx.CallbackQuery.From.Id
		log.Println("Cannot get Callback ID from database:", err)
		_, err = ctx.EffectiveChat.SendMessage(bot, t(uid, "err.internal"), nil)
		return err
	}
	// Check the link
	link, exists := cachedData.Links[data.LinkKey]
	if !exists {
		uid := ctx.CallbackQuery.From.Id
		_, err = ctx.EffectiveChat.SendMessage(bot, t(uid, "err.resend_link"), nil)
		return err
	}
	dim := reddit.Dimension{
		Width:  link.Width,
		Height: link.Height,
	}
	// Check the media type
	switch cachedData.Type {
	case reddit.FetchResultMediaTypeGif:
		return c.handleGifUpload(bot, link.Link, cachedData.Title, cachedData.ThumbnailLink, cachedData.PostLink, cachedData.Description, dim, ctx.EffectiveChat.Id)
	case reddit.FetchResultMediaTypePhoto:
		return c.handlePhotoUpload(bot, link.Link, cachedData.Title, cachedData.ThumbnailLink, cachedData.PostLink, cachedData.Description, ctx.EffectiveChat.Id, data.Mode == CallbackButtonDataModePhoto)
	case reddit.FetchResultMediaTypeVideo:
		if data.LinkKey == cachedData.AudioIndex {
			return c.handleAudioUpload(bot, link.Link, cachedData.Title, cachedData.PostLink, cachedData.Description, cachedData.Duration, ctx.EffectiveChat.Id)
		} else {
			audioURL := cachedData.Links[cachedData.AudioIndex]
			return c.handleVideoUpload(bot, link.Link, audioURL.Link, cachedData.Title, cachedData.ThumbnailLink, cachedData.PostLink, cachedData.Description, dim, cachedData.Duration, ctx.EffectiveChat.Id)
		}
	}
	// What
	panic("Unknown media type: " + strconv.Itoa(int(cachedData.Type)))
}

type settingsCallbackData struct {
	Kind   string `json:"k"`
	Action string `json:"a"`
	Value  string `json:"v"`
}

func (s *settingsCallbackData) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

func NewSettingsCallbackData(kind string, action string, value string) *settingsCallbackData {
	return &settingsCallbackData{
		Kind:   kind,
		Action: action,
		Value:  value,
	}
}
