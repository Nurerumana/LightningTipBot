package telegram

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LightningTipBot/LightningTipBot/internal/errors"
	"github.com/LightningTipBot/LightningTipBot/internal/satdress"

	"github.com/LightningTipBot/LightningTipBot/internal"

	log "github.com/sirupsen/logrus"

	"github.com/LightningTipBot/LightningTipBot/internal/i18n"
	"github.com/LightningTipBot/LightningTipBot/internal/lnbits"
	"github.com/LightningTipBot/LightningTipBot/internal/runtime"
	"github.com/LightningTipBot/LightningTipBot/internal/str"
	"github.com/skip2/go-qrcode"
	tb "gopkg.in/lightningtipbot/telebot.v2"
)

type InvoiceEventCallback map[int]func(*InvoiceEvent)

var InvoiceCallback InvoiceEventCallback

func initInvoiceEventCallbacks(bot *TipBot) {
	InvoiceCallback = InvoiceEventCallback{
		InvoiceCallbackGeneric:         bot.notifyInvoiceReceivedEvent,
		InvoiceCallbackInlineReceive:   bot.inlineReceiveEvent,
		InvoiceCallbackLNURLPayReceive: bot.lnurlReceiveEvent,
	}
}

type InvoiceEventKey int

const (
	InvoiceCallbackGeneric = iota + 1
	InvoiceCallbackInlineReceive
	InvoiceCallbackLNURLPayReceive
)

type Invoice struct {
	PaymentHash    string `json:"payment_hash"`
	PaymentRequest string `json:"payment_request"`
	Amount         int64  `json:"amount"`
	Memo           string `json:"memo"`
}
type InvoiceEvent struct {
	*Invoice
	User           *lnbits.User `json:"user"`
	Message        *tb.Message  `json:"message"`
	InvoiceMessage *tb.Message  `json:"invoice_message"`
	LanguageCode   string       `json:"languagecode"`
	Callback       int          `json:"func"`
	CallbackData   string       `json:"callbackdata"`
}

func (invoiceEvent InvoiceEvent) Key() string {
	return fmt.Sprintf("invoice:%s", invoiceEvent.PaymentHash)
}

func helpInvoiceUsage(ctx context.Context, errormsg string) string {
	if len(errormsg) > 0 {
		return fmt.Sprintf(Translate(ctx, "invoiceHelpText"), fmt.Sprintf("%s", errormsg))
	} else {
		return fmt.Sprintf(Translate(ctx, "invoiceHelpText"), "")
	}
}

func (bot *TipBot) invoiceHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	// check and print all commands
	bot.anyTextHandler(ctx, m)
	user := LoadUser(ctx)
	if user.Wallet == nil {
		return ctx, errors.Create(errors.UserNoWalletError)
	}
	userStr := GetUserStr(user.Telegram)
	if m.Chat.Type != tb.ChatPrivate {
		// delete message
		bot.tryDeleteMessage(m)
		return ctx, errors.Create(errors.NoPrivateChatError)
	}
	// if no amount is in the command, ask for it
	amount, err := decodeAmountFromCommand(m.Text)
	if (err != nil || amount < 1) && m.Chat.Type == tb.ChatPrivate {
		// // no amount was entered, set user state and ask fo""r amount
		_, err = bot.askForAmount(ctx, "", "CreateInvoiceState", 0, 0, m.Text)
		return ctx, err
	}

	// check for memo in command
	memo := "Powered by @LightningTipBot"
	if len(strings.Split(m.Text, " ")) > 2 {
		memo = GetMemoFromCommand(m.Text, 2)
		tag := " (@LightningTipBot)"
		memoMaxLen := 159 - len(tag)
		if len(memo) > memoMaxLen {
			memo = memo[:memoMaxLen-len(tag)]
		}
		memo = memo + tag
	}

	creatingMsg := bot.trySendMessageEditable(m.Sender, Translate(ctx, "lnurlGettingUserMessage"))
	log.Debugf("[/invoice] Creating invoice for %s of %d sat.", userStr, amount)
	invoice, err := bot.createInvoiceWithEvent(ctx, user, amount, memo, InvoiceCallbackGeneric, "")
	if err != nil {
		errmsg := fmt.Sprintf("[/invoice] Could not create an invoice: %s", err.Error())
		bot.tryEditMessage(creatingMsg, Translate(ctx, "errorTryLaterMessage"))
		log.Errorln(errmsg)
		return ctx, err
	}

	// create qr code
	qr, err := qrcode.Encode(invoice.PaymentRequest, qrcode.Medium, 256)
	if err != nil {
		errmsg := fmt.Sprintf("[/invoice] Failed to create QR code for invoice: %s", err.Error())
		bot.tryEditMessage(creatingMsg, Translate(ctx, "errorTryLaterMessage"))
		log.Errorln(errmsg)
		return ctx, err
	}

	// deleting messages will delete the main menu.
	//bot.tryDeleteMessage(creatingMsg)

	// send the invoice data to user
	bot.trySendMessage(m.Sender, &tb.Photo{File: tb.File{FileReader: bytes.NewReader(qr)}, Caption: fmt.Sprintf("`%s`", invoice.PaymentRequest)})
	log.Printf("[/invoice] Incvoice created. User: %s, amount: %d sat.", userStr, amount)
	return ctx, nil
}

func (bot *TipBot) createInvoiceWithEvent(ctx context.Context, user *lnbits.User, amount int64, memo string, callback int, callbackData string) (InvoiceEvent, error) {
	invoice, err := user.Wallet.Invoice(
		lnbits.InvoiceParams{
			Out:     false,
			Amount:  int64(amount),
			Memo:    memo,
			Webhook: internal.Configuration.Lnbits.WebhookServer},
		bot.Client)
	if err != nil {
		errmsg := fmt.Sprintf("[/invoice] Could not create an invoice: %s", err.Error())
		log.Errorln(errmsg)
		return InvoiceEvent{}, err
	}
	invoiceEvent := InvoiceEvent{
		Invoice: &Invoice{PaymentHash: invoice.PaymentHash,
			PaymentRequest: invoice.PaymentRequest,
			Amount:         amount,
			Memo:           memo},
		User:         user,
		Callback:     callback,
		CallbackData: callbackData,
		LanguageCode: ctx.Value("publicLanguageCode").(string),
	}
	// save invoice struct for later use
	runtime.IgnoreError(bot.Bunt.Set(invoiceEvent))
	return invoiceEvent, nil
}

func (bot *TipBot) notifyInvoiceReceivedEvent(invoiceEvent *InvoiceEvent) {
	// do balance check for keyboard update
	_, err := bot.GetUserBalance(invoiceEvent.User)
	if err != nil {
		errmsg := fmt.Sprintf("could not get balance of user %s", GetUserStr(invoiceEvent.User.Telegram))
		log.Errorln(errmsg)
	}

	bot.trySendMessage(invoiceEvent.User.Telegram, fmt.Sprintf(i18n.Translate(invoiceEvent.User.Telegram.LanguageCode, "invoiceReceivedMessage"), invoiceEvent.Amount))
}

type LNURLInvoice struct {
	*Invoice
	Comment   string       `json:"comment"`
	User      *lnbits.User `json:"user"`
	CreatedAt time.Time    `json:"created_at"`
	Paid      bool         `json:"paid"`
	PaidAt    time.Time    `json:"paid_at"`
	From      string       `json:"from"`
}

func (lnurlInvoice LNURLInvoice) Key() string {
	return fmt.Sprintf("lnurl-p:%s", lnurlInvoice.PaymentHash)
}

func (bot *TipBot) lnurlReceiveEvent(invoiceEvent *InvoiceEvent) {
	bot.notifyInvoiceReceivedEvent(invoiceEvent)
	tx := &LNURLInvoice{Invoice: &Invoice{PaymentHash: invoiceEvent.PaymentHash}}
	err := bot.Bunt.Get(tx)
	log.Debugf("[lnurl-p] Received invoice for %s of %d sat.", GetUserStr(invoiceEvent.User.Telegram), tx.Amount)
	if err == nil {
		if len(tx.Comment) > 0 {
			if len(tx.From) == 0 {
				bot.trySendMessage(tx.User.Telegram, fmt.Sprintf("✉️ %s", str.MarkdownEscape(tx.Comment)))
			} else {
				bot.trySendMessage(tx.User.Telegram, fmt.Sprintf("✉️ From `%s`: %s", tx.From, str.MarkdownEscape(tx.Comment)))
			}
		} else if len(tx.From) > 0 {
			bot.trySendMessage(tx.User.Telegram, fmt.Sprintf("From `%s`", str.MarkdownEscape(tx.From)))
		}
	}
}

// TODO -- move somewhere else
func (bot *TipBot) invHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	// check and print all commands
	bot.anyTextHandler(ctx, m)
	user := LoadUser(ctx)
	if user.Wallet == nil {
		return ctx, errors.Create(errors.UserNoWalletError)
	}

	// get invoice from user node
	pr, err := satdress.GetInvoice(
		satdress.Params{
			Backend: satdress.LNDParams{
				Cert:     "2d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d494943496a4343416369674177494241674952414f68774131694b486b4d704b3654734a797836794b4177436759494b6f5a497a6a3045417749774d4445660a4d4230474131554543684d576247356b494746316447396e5a57356c636d46305a575167593256796444454e4d4173474131554541784d455a4746325a5441650a467730794d5445794d4463794d6a51314d544e61467730794d7a41794d4445794d6a51314d544e614d444178487a416442674e5642416f54466d78755a4342680a645852765a3256755a584a686447566b49474e6c636e51784454414c42674e5642414d5442475268646d55775754415442676371686b6a4f50514942426767710a686b6a4f50514d4242774e4341415350514146462f586838655666496d43414f7a6a456d57596d2f736470632b616a535a50654245333342305369787a3433350a30427976344e317033396d54527a4f783848647332777562326e6d505958636d2b6263506f3448434d49472f4d41344741315564447745422f775145417749430a7044415442674e56485355454444414b4267677242674546425163444154415042674e5648524d4241663845425441444151482f4d42304741315564446751570a424254734732594c32666a6744467954366c2b474f777671762b38634d44426f42674e5648524545595442666767526b59585a6c67676c7362324e68624768760a6333534342475268646d574344584276624746794c5734794c575268646d57434248567561586943436e56756158687759574e725a58534342324a315a6d4e760a626d36484248384141414748454141414141414141414141414141414141414141414748424b775741415977436759494b6f5a497a6a304541774944534141770a52514968414f6437436c716e4a3258735571716b5953756e4937777147736e4b596d57334668353045765877775a44394169424e4c6e575a4342416a343664780a726f5a392f435563595a78754756432f6d666b2b38325a2b5073777977413d3d0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a",
				Host:     "https://127.0.0.1:8082",
				Macaroon: "AgEDbG5kAlgDChBdAmYXLaSkDZ+jUNXkXedBEgEwGhYKB2FkZHJlc3MSBHJlYWQSBXdyaXRlGhcKCGludm9pY2VzEgRyZWFkEgV3cml0ZRoPCgdvbmNoYWluEgRyZWFkAAAGIBevooGKO6xo8c0Q2fj14DYYx95b+jah0rfMj+JJ6pf3",
			},
			Msatoshi: 1000,
		},
	)
	if err != nil {
		log.Errorln(err.Error())
		return ctx, err
	}

	log.Infof(pr)

	return ctx, nil
}
