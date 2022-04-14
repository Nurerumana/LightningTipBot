package telegram

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/LightningTipBot/LightningTipBot/internal/runtime"
	"github.com/LightningTipBot/LightningTipBot/internal/telegram/intercept"
	tb "gopkg.in/lightningtipbot/telebot.v3"

	"github.com/LightningTipBot/LightningTipBot/internal/lnbits"
	"github.com/LightningTipBot/LightningTipBot/internal/satdress"
	"github.com/LightningTipBot/LightningTipBot/internal/storage"
	"github.com/eko/gocache/store"
	log "github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
)

var (
	checkingInvoiceMessage    = "⏳ Checking invoice on your node..."
	invoiceNotSettledMessage  = "❌ Invoice has not settled yet."
	checkInvoiceButtonMessage = "🔄 Check invoice"
	invoiceSettledMessage     = "✅ *Invoice settled.*"
	satdressCheckInvoicenMenu = &tb.ReplyMarkup{ResizeKeyboard: true}
	btnSatdressCheckInvoice   = satdressCheckInvoicenMenu.Data(checkInvoiceButtonMessage, "satdress_check_invoice")
)

// todo -- rename to something better like parse node settings or something
func parseUserSettingInput(ctx intercept.Context, m *tb.Message) (satdress.LNDParams, error) {
	// input is "/node add <Host> <Macaroon> <Cert>"
	params := satdress.LNDParams{}
	splits := strings.Split(m.Text, " ")
	splitlen := len(splits)
	if splitlen < 4 || splitlen > 5 {
		return params, fmt.Errorf("wrong format! Use <Host> <Macaroon> <Cert>")
	}
	host := splits[2]
	macaroon := splits[3]
	cert := splits[4]

	hostsplit := strings.Split(host, ".")
	if len(hostsplit) == 0 {
		return params, fmt.Errorf("host wrong format")
	}
	pem := parseCertificateToPem(cert)
	return satdress.LNDParams{
		Cert:       pem,
		Host:       host,
		Macaroon:   macaroon,
		CertString: string(pem),
	}, nil
}

func (bot *TipBot) getNodeHandler(ctx intercept.Context) (intercept.Context, error) {
	m := ctx.Message()
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		log.Infof("Could not get user settings for user %s", GetUserStr(user.Telegram))
		return ctx, err
	}
	node_info_str := "*Host:*\n`%s`\n*Macaroon:*\n`%s`\n*Cert:*\n`%s`"
	if user.Settings != nil && user.Settings.Node.LNDParams != nil {
		node_info_str_filled := fmt.Sprintf(node_info_str, user.Settings.Node.LNDParams.Host, user.Settings.Node.LNDParams.Macaroon, user.Settings.Node.LNDParams.CertString)
		resp_str := fmt.Sprintf("ℹ️ *Your node information.*\n\n%s", node_info_str_filled)
		bot.trySendMessage(m.Sender, resp_str)
	} else {
		bot.trySendMessage(m.Sender, "You did not register a node yet.")
	}
	return ctx, nil
}

func (bot *TipBot) nodeHandler(ctx intercept.Context) (intercept.Context, error) {
	m := ctx.Message()
	splits := strings.Split(m.Text, " ")
	if len(splits) == 1 {
		return bot.getNodeHandler(ctx)
	} else if len(splits) > 1 {
		if splits[1] == "invoice" {
			return bot.invHandler(ctx)
		}
		if splits[1] == "add" {
			return bot.registerNodeHandler(ctx)
		}
		if splits[1] == "check" {
			return bot.satdressCheckInvoiceHandler(ctx)
		}
		if splits[1] == "proxy" {
			return bot.satdressProxyHandler(ctx)
		}
	}
	return ctx, nil
}

func (bot *TipBot) registerNodeHandler(ctx intercept.Context) (intercept.Context, error) {
	m := ctx.Message()
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		return ctx, err
	}
	lndparams, err := parseUserSettingInput(ctx, m)
	if err != nil {
		return ctx, err
	}
	user.Settings.Node.LNDParams = &lndparams
	user.Settings.Node.NodeType = "lnd"
	err = UpdateUserRecord(user, *bot)
	if err != nil {
		log.Errorf("[registerNodeHandler] could not update record of user %s: %v", GetUserStr(user.Telegram), err)
		return ctx, err
	}

	node_info_str := "*Host:*\n`%s`\n*Macaroon:*\n`%s`\n*Cert:*\n`%s`"
	node_info_str_filled := fmt.Sprintf(node_info_str, lndparams.Host, lndparams.Macaroon, lndparams.Cert)
	resp_str := fmt.Sprintf("✅ *Node added.*\n\n%s", node_info_str_filled)
	bot.trySendMessage(m.Sender, resp_str)
	return ctx, nil
}

func (bot *TipBot) invHandler(ctx intercept.Context) (intercept.Context, error) {
	m := ctx.Message()
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		return ctx, err
	}
	if user.Settings == nil || user.Settings.Node.LNDParams == nil {
		bot.trySendMessage(m.Sender, "You did not register a node yet.")
		return ctx, fmt.Errorf("node of user %s not registered", GetUserStr(user.Telegram))
	}

	var amount int64
	if amount_str, err := getArgumentFromCommand(m.Text, 2); err == nil {
		amount, err = getAmount(amount_str)
		if err != nil {
			return ctx, err
		}
	}

	// get invoice from user's node
	getInvoiceParams, err := satdress.GetInvoice(
		satdress.GetInvoiceParams{
			Backend: satdress.LNDParams{
				Cert:     []byte(user.Settings.Node.LNDParams.CertString),
				Host:     user.Settings.Node.LNDParams.Host,
				Macaroon: user.Settings.Node.LNDParams.Macaroon,
			},
			Msatoshi: amount * 1000,
		},
	)
	if err != nil {
		log.Errorln(err.Error())
		return ctx, err
	}
	// bot.trySendMessage(m.Sender, fmt.Sprintf("PR: `%s`\n\nHash: `%s`\n\nStatus: `%s`", getInvoiceParams.PR, string(getInvoiceParams.Hash), getInvoiceParams.Status))

	// create qr code
	qr, err := qrcode.Encode(getInvoiceParams.PR, qrcode.Medium, 256)
	if err != nil {
		errmsg := fmt.Sprintf("[/invoice] Failed to create QR code for invoice: %s", err.Error())
		bot.trySendMessage(user.Telegram, Translate(ctx, "errorTryLaterMessage"))
		log.Errorln(errmsg)
		return ctx, err
	}
	bot.trySendMessage(m.Sender, &tb.Photo{File: tb.File{FileReader: bytes.NewReader(qr)}, Caption: fmt.Sprintf("`%s`", getInvoiceParams.PR)})

	// add the getInvoiceParams to cache to check it later
	bot.Cache.Set(fmt.Sprintf("invoice:%d", user.Telegram.ID), getInvoiceParams, &store.Options{Expiration: 24 * time.Hour})

	// check if invoice settles
	return bot.satdressCheckInvoiceHandler(ctx)
}

func (bot *TipBot) satdressCheckInvoiceHandler(ctx intercept.Context) (intercept.Context, error) {
	tgUser := LoadUser(ctx).Telegram
	user, err := GetLnbitsUserWithSettings(tgUser, *bot)
	if err != nil {
		return ctx, err
	}

	// get the getInvoiceParams from cache
	log.Debugf("[Cache] Getting key: %s", fmt.Sprintf("invoice:%d", user.Telegram.ID))
	getInvoiceParamsInterface, err := bot.Cache.Get(fmt.Sprintf("invoice:%d", user.Telegram.ID))
	if err != nil {
		log.Errorf("[satdressCheckInvoiceHandler] UserID: %d,  %s", user.Telegram.ID, err.Error())
		return ctx, err
	}
	getInvoiceParams := getInvoiceParamsInterface.(satdress.CheckInvoiceParams)

	// check the invoice

	// check if there is an invoice check message in cache already
	check_message_interface, err := bot.Cache.Get(fmt.Sprintf("invoice:msg:%s", getInvoiceParams.Hash))
	var check_message *tb.Message
	if err != nil {
		// send a new message if there isn't one in the cache
		check_message = bot.trySendMessageEditable(tgUser, checkingInvoiceMessage)
	} else {
		check_message = check_message_interface.(*tb.Message)
		check_message, err = bot.tryEditMessage(check_message, checkingInvoiceMessage)
		if err != nil {
			log.Errorf("[satdressCheckInvoiceHandler] UserID: %d,  %s", user.Telegram.ID, err.Error())
		}
	}

	// save it in the cache for another call later
	bot.Cache.Set(fmt.Sprintf("invoice:msg:%s", getInvoiceParams.Hash), check_message, &store.Options{Expiration: 24 * time.Hour})

	deadLineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*10))
	runtime.NewRetryTicker(deadLineCtx, "node_invoice_check", runtime.WithRetryDuration(5*time.Second)).Do(func() {
		// get invoice from user's node
		getInvoiceParams, err = satdress.CheckInvoice(getInvoiceParams)
		if err != nil {
			log.Errorln(err.Error())
			return
		}
		if getInvoiceParams.Status == "SETTLED" {
			bot.tryEditMessage(check_message, invoiceSettledMessage)
			cancel()
		}

	}, func() {
		// cancel
	},
		func() {
			// deadline
			bot.tryEditMessage(check_message, invoiceNotSettledMessage,
				&tb.ReplyMarkup{
					InlineKeyboard: [][]tb.InlineButton{
						{tb.InlineButton{Text: checkInvoiceButtonMessage, Unique: "satdress_check_invoice"}},
					},
				})
		},
	)

	return ctx, nil
}

func parseCertificateToPem(cert string) []byte {
	block, _ := pem.Decode([]byte(cert))
	if block != nil {
		// already PEM
		return []byte(cert)
	} else {
		var dec []byte

		dec, err := hex.DecodeString(cert)
		if err != nil {
			// not HEX
			dec, err = base64.StdEncoding.DecodeString(cert)
			if err != nil {
				// not base54, we have a problem huston
				return nil
			}
		}
		if block, _ := pem.Decode(dec); block != nil {
			return dec
		}
		// decoding went wrong
		return nil
	}
}

func (bot *TipBot) satdressProxyHandler(ctx intercept.Context) (intercept.Context, error) {
	m := ctx.Message()
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		return ctx, err
	}
	if user.Settings == nil || user.Settings.Node.LNDParams == nil {
		bot.trySendMessage(user.Telegram, "You did not register a node yet.")
		log.Errorf("node of user %s not registered", GetUserStr(user.Telegram))
		return ctx, fmt.Errorf("no node settings.")
	}

	var amount int64
	if amount_str, err := getArgumentFromCommand(m.Text, 2); err == nil {
		amount, err = getAmount(amount_str)
		if err != nil {
			return ctx, err
		}
	}

	memo := "Proxy relay invoice"
	invoice, err := bot.createInvoiceWithEvent(ctx, user, amount, memo, InvoiceCallbackSatdressProxy, "")
	if err != nil {
		errmsg := fmt.Sprintf("[/invoice] Could not create an invoice: %s", err.Error())
		bot.trySendMessage(user.Telegram, Translate(ctx, "errorTryLaterMessage"))
		log.Errorln(errmsg)
		return ctx, err
	}

	// create qr code
	qr, err := qrcode.Encode(invoice.PaymentRequest, qrcode.Medium, 256)
	if err != nil {
		errmsg := fmt.Sprintf("[/invoice] Failed to create QR code for invoice: %s", err.Error())
		bot.trySendMessage(user.Telegram, Translate(ctx, "errorTryLaterMessage"))
		log.Errorln(errmsg)
		return ctx, err
	}
	bot.trySendMessage(m.Sender, &tb.Photo{File: tb.File{FileReader: bytes.NewReader(qr)}, Caption: fmt.Sprintf("`%s`", invoice.PaymentRequest)})
	return ctx, nil
}

func (bot *TipBot) satdressProxyRelayPaymentHandler(event Event) {
	invoiceEvent := event.(*InvoiceEvent)
	user := invoiceEvent.User
	if user.Settings == nil || user.Settings.Node.LNDParams == nil {
		bot.trySendMessage(user.Telegram, "You did not register a node yet.")
		log.Errorf("node of user %s not registered", GetUserStr(user.Telegram))
		return
	}

	bot.notifyInvoiceReceivedEvent(invoiceEvent)

	// now relay the payment to the user's node
	var amount int64 = invoiceEvent.Amount

	// get invoice from user's node
	getInvoiceParams, err := satdress.GetInvoice(
		satdress.GetInvoiceParams{
			Backend: satdress.LNDParams{
				Cert:     []byte(user.Settings.Node.LNDParams.CertString),
				Host:     user.Settings.Node.LNDParams.Host,
				Macaroon: user.Settings.Node.LNDParams.Macaroon,
			},
			Msatoshi: amount * 1000,
		},
	)
	if err != nil {
		log.Errorln(err.Error())
		return
	}

	bot.trySendMessage(user.Telegram, fmt.Sprintf("PR: `%s`\n\nHash: `%s`\n\nStatus: `%s`", getInvoiceParams.PR, string(getInvoiceParams.Hash), getInvoiceParams.Status))

	// pay invoice
	invoice, err := user.Wallet.Pay(lnbits.PaymentParams{Out: true, Bolt11: getInvoiceParams.PR}, bot.Client)
	if err != nil {
		errmsg := fmt.Sprintf("[/pay] Could not pay invoice of %s: %s", GetUserStr(user.Telegram), err)
		// err = fmt.Errorf(i18n.Translate(payData.LanguageCode, "invoiceUndefinedErrorMessage"))
		// bot.tryEditMessage(c.Message, fmt.Sprintf(i18n.Translate(payData.LanguageCode, "invoicePaymentFailedMessage"), err.Error()), &tb.ReplyMarkup{})
		// verbose error message, turned off for now
		// if len(err.Error()) == 0 {
		// 	err = fmt.Errorf(i18n.Translate(payData.LanguageCode, "invoiceUndefinedErrorMessage"))
		// }
		// bot.tryEditMessage(c.Message, fmt.Sprintf(i18n.Translate(payData.LanguageCode, "invoicePaymentFailedMessage"), str.MarkdownEscape(err.Error())), &tb.ReplyMarkup{})
		log.Errorln(errmsg)
		return
	}

	// object that holds all information about the send payment
	id := fmt.Sprintf("proxypay:%d:%d:%s", user.Telegram.ID, amount, RandStringRunes(8))

	payData := &PayData{
		Base:    storage.New(storage.ID(id)),
		From:    user,
		Invoice: invoice.PaymentRequest,
		Hash:    invoice.PaymentHash,
		Amount:  amount,
	}
	// add result to persistent struct
	runtime.IgnoreError(payData.Set(payData, bot.Bunt))

	// add the getInvoiceParams to cache to check it later
	bot.Cache.Set(fmt.Sprintf("invoice:%d", user.Telegram.ID), getInvoiceParams, &store.Options{Expiration: 24 * time.Hour})

	time.Sleep(time.Second)

	getInvoiceParams, err = satdress.CheckInvoice(getInvoiceParams)
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	bot.trySendMessage(user.Telegram, fmt.Sprintf("PR: `%s`\n\nHash: `%s`\n\nStatus: `%s`", getInvoiceParams.PR, string(getInvoiceParams.Hash), getInvoiceParams.Status))

	return
}
