package telegram

import (
	"context"
	"fmt"
	"github.com/LightningTipBot/LightningTipBot/internal/log"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/LightningTipBot/LightningTipBot/internal/telegram/intercept"

	"github.com/LightningTipBot/LightningTipBot/internal/errors"

	"github.com/LightningTipBot/LightningTipBot/internal/str"

	"github.com/LightningTipBot/LightningTipBot/internal/lnbits"
	logrus "github.com/sirupsen/logrus"
	tb "gopkg.in/lightningtipbot/telebot.v3"
)

// PLEASE DO NOT CHANGE THE CODE IN THIS FILE
// YOU MIGHT BREAK DONATIONS TO THE ORIGINAL PROJECT
// THE DEVELOPMENT OF LIGHTNINGTIPBOT RELIES ON DONATIONS
// IF YOU USE THIS PROJECT, LEAVE THIS CODE ALONE

var (
	donationEndpoint string
)

func helpDonateUsage(ctx context.Context, errormsg string) string {
	if len(errormsg) > 0 {
		return fmt.Sprintf(Translate(ctx, "donateHelpText"), fmt.Sprintf("%s", errormsg))
	} else {
		return fmt.Sprintf(Translate(ctx, "donateHelpText"), "")
	}
}

func (bot TipBot) donationHandler(ctx intercept.Context) (intercept.Context, error) {
	// check and print all commands
	logFields := logrus.Fields{
		"module": "telegram",
		"func":   "donationHandler"}
	m := ctx.Message()
	bot.anyTextHandler(ctx)
	user := LoadUser(ctx)
	if user.Wallet == nil {
		return ctx, errors.Create(errors.UserNoWalletError)
	}
	// if no amount is in the command, ask for it
	amount, err := decodeAmountFromCommand(m.Text)
	if (err != nil || amount < 1) && m.Chat.Type == tb.ChatPrivate {
		// // no amount was entered, set user state and ask for amount
		_, err = bot.askForAmount(ctx, "", "CreateDonationState", 0, 0, m.Text)
		return ctx, err
	}

	// command is valid
	msg := bot.trySendMessageEditable(m.Chat, Translate(ctx, "donationProgressMessage"))
	// get invoice
	resp, err := http.Get(fmt.Sprintf(donationEndpoint, amount, user.GetUserStr(), GetUserStr(bot.Telegram.Me)))
	if err != nil {
		log.WithObjects(user, err, ctx).WithFields(logFields).Errorln("could not GET donation endpoint")
		bot.tryEditMessage(msg, Translate(ctx, "donationErrorMessage"))
		return ctx, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithObjects(user, err, ctx).WithFields(logFields).Errorln("could not read body")
		bot.tryEditMessage(msg, Translate(ctx, "donationErrorMessage"))
		return ctx, err
	}

	// send donation invoice
	// user := LoadUser(ctx)
	// bot.trySendMessage(user.Telegram, string(body))
	_, err = user.Wallet.Pay(lnbits.PaymentParams{Out: true, Bolt11: string(body)}, bot.Client)
	if err != nil {
		log.WithObjects(ctx, user, err).WithFields(logFields).Errorln("donation failed")
		bot.tryEditMessage(msg, Translate(ctx, "donationErrorMessage"))
		return ctx, err
	}
	// hotfix because the edit doesn't work!
	// todo: fix edit
	// bot.tryEditMessage(msg, Translate(ctx, "donationSuccess"))
	bot.tryDeleteMessage(msg)
	bot.trySendMessage(m.Chat, Translate(ctx, "donationSuccess"))
	return ctx, nil
}

func init() {
	var sb strings.Builder
	rot := []byte{117, 103, 103, 99, 102, 58, 47, 47, 121, 97, 46, 103, 118, 99, 102, 47, 113, 98, 97, 110, 103, 114, 47, 37, 113, 63, 115, 101, 98, 122, 61, 37, 102, 38, 111, 98, 103, 61, 37, 102}
	_, err := io.Copy(&sb, rot13Reader{strings.NewReader(string(rot))})
	if err != nil {
		panic(err)
	}
	donationEndpoint = sb.String()
}

type rot13Reader struct {
	r io.Reader
}

func (rot13 rot13Reader) Read(b []byte) (int, error) {
	n, err := rot13.r.Read(b)
	for i := 0; i < n; i++ {
		switch {
		case b[i] >= 65 && b[i] <= 90:
			if b[i] <= 77 {
				b[i] = b[i] + 13
			} else {
				b[i] = b[i] - 13
			}
		case b[i] >= 97 && b[i] <= 122:
			if b[i] <= 109 {
				b[i] = b[i] + 13
			} else {
				b[i] = b[i] - 13
			}
		}
	}
	return n, err
}

var d = []byte{71, 117, 110, 97, 120, 32, 108, 98, 104, 33, 32, 86, 39, 122, 32, 101, 98, 104, 103, 118, 97, 116, 32, 103, 117, 118, 102, 32, 113, 98, 97, 110, 103, 118, 98, 97, 32, 103, 98, 32, 89, 118, 116, 117, 103, 97, 118, 97, 116, 71, 118, 99, 79, 98, 103, 64, 121, 97, 46, 103, 118, 99, 102, 46}

func (bot TipBot) parseDonationCommandHandler(ctx intercept.Context) error {
	m := ctx.Message()
	arg := ""
	if strings.HasPrefix(strings.ToLower(m.Text), "/send") {
		arg, _ = getArgumentFromCommand(m.Text, 2)
		if arg != "@"+bot.Telegram.Me.Username {
			return fmt.Errorf("err")
		}
	}
	if strings.HasPrefix(strings.ToLower(m.Text), "/tip") {
		arg = GetUserStr(m.ReplyTo.Sender)
		if arg != "@"+bot.Telegram.Me.Username {
			return fmt.Errorf("err")
		}
	}
	if arg == "@LightningTipBot" || len(arg) < 1 {
		return fmt.Errorf("err")
	}

	amount, err := decodeAmountFromCommand(m.Text)
	if err != nil {
		return err
	}

	var sb strings.Builder
	_, err = io.Copy(&sb, rot13Reader{strings.NewReader(string(d))})
	if err != nil {
		panic(err)
	}
	donationInterceptMessage := sb.String()

	bot.trySendMessage(m.Sender, str.MarkdownEscape(donationInterceptMessage))
	m.Text = fmt.Sprintf("/donate %d", amount)
	ctx, err = bot.donationHandler(ctx)
	// returning nil here will abort the parent ctx (/pay or /tip)
	return err
}
