// ratelimiter Project
// Copyright (C) 2021~2022 ALiwoto and other Contributors
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of the source code.

package tests

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ALiwoto/ratelimiter"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

const TEST_TIME_OUT = 29 * time.Minute

var testTimeout time.Duration

func TestRateLimiter(t *testing.T) {
	var token string
	if os.PathSeparator == '/' {
		token = os.Getenv("BOT_TOKEN")
	} else {
		token = os.Getenv("BOT_TOKEN_WINDOWS")
	}

	if token == "" {
		log.Println("trying to load the token from file")
		f, err := os.Open("config")
		if err != nil {
			log.Printf("failed to load the config file: %v\n", err)
			return
		}

		var b []byte
		b, err = io.ReadAll(f)
		if err != nil {
			t.Errorf("failed to load the config file: %v", err)
			return
		} else if len(b) == 0 {
			t.Error("token loaded from the config file is empty")
			return
		}

		token = string(b)
	}

	token = strings.TrimSpace(token)
	log.Println("token is: ", token)

	timeoutStr := os.Getenv("TIME_OUT")
	if timeoutStr == "" {
		testTimeout = TEST_TIME_OUT
	} else {
		tInt, err := strconv.Atoi(timeoutStr)
		if err != nil {
			testTimeout = TEST_TIME_OUT
		} else {
			testTimeout = time.Duration(tInt) * time.Second
		}
	}

	bot, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{},
			DefaultRequestOpts: &gotgbot.RequestOpts{
				Timeout: 6 * gotgbot.DefaultTimeout,
			},
		},
	})
	if err != nil {
		t.Errorf("failed to create a new bot instance: %v", err)
		return
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{})
	updater := ext.NewUpdater(dispatcher, nil)
	loadHandlers(dispatcher)

	err = updater.StartPolling(bot, &ext.PollingOpts{
		DropPendingUpdates: true,
	})

	if err != nil {
		// "Failed to start polling due to %s"
		t.Errorf("failed to start polling due to : %v", err)
		return
	}

	time.Sleep(testTimeout)

}

func loadHandlers(d *ext.Dispatcher) {
	limiter := ratelimiter.NewLimiter(d, &ratelimiter.LimiterConfig{
		ConsiderChannel:  false,
		ConsiderUser:     true,
		ConsiderEdits:    false,
		IgnoreMediaGroup: true,
		TextOnly:         false,
		HandlerGroups:    []int{0, 1, 2},
	})
	limiter.SetTriggerFuncs(limitedTrigger)
	limiter.Start()

	msgHandler := handlers.NewMessage(func(msg *gotgbot.Message) bool {
		return true
	}, func(b *gotgbot.Bot, ctx *ext.Context) error {
		ctx.EffectiveMessage.Reply(b, "received text: "+ctx.EffectiveMessage.Text,
			&gotgbot.SendMessageOpts{})
		return nil
	})

	d.AddHandler(msgHandler)
}

func limitedTrigger(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	msg.Reply(b, "you have been limited!",
		&gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				AllowSendingWithoutReply: msg.Chat.Type == "private",
			},
		})

	return nil
}
