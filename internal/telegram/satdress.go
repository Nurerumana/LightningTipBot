package telegram

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/LightningTipBot/LightningTipBot/internal/runtime"
	"strings"
	"time"

	"github.com/LightningTipBot/LightningTipBot/internal/satdress"
	"github.com/eko/gocache/store"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/lightningtipbot/telebot.v2"
)

// todo -- rename to something better like parse node settings or something
func parseUserSettingInput(ctx context.Context, m *tb.Message) (satdress.LNDParams, error) {
	// input is "/node add <Host> <Macaroon> <Cert>"
	params := satdress.LNDParams{}
	splits := strings.Split(m.Text, " ")
	splitlen := len(splits)
	if splitlen < 4 || splitlen > 5 {
		return params, fmt.Errorf("Wrong format! Use <Host> <Macaroon> <Cert>")
	}
	host := splits[2]
	macaroon := splits[3]
	cert := splits[4]

	hostsplit := strings.Split(host, ".")
	if len(hostsplit) == 0 {
		return params, fmt.Errorf("Host wrong format")
	}
	pem := parseCertificateToPem(cert)
	return satdress.LNDParams{
		Cert:       pem,
		Host:       host,
		Macaroon:   macaroon,
		CertString: string(pem),
	}, nil
}

func (bot *TipBot) getNodeHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		log.Infof("Could not get user settings for user %s", GetUserStr(user.Telegram))
		return ctx, err
	}
	node_info_str := "*Host:*\n`%s`\n*Macaroon:*\n`%s`\n*Cert:*\n`%s`"
	if user.Settings != nil {
		node_info_str_filled := fmt.Sprintf(node_info_str, user.Settings.LNDParams.Host, user.Settings.LNDParams.Macaroon, user.Settings.LNDParams.Cert)
		resp_str := fmt.Sprintf("ℹ️ *Your node information.*\n\n%s", node_info_str_filled)
		bot.trySendMessage(m.Sender, resp_str)
	} else {
		bot.trySendMessage(m.Sender, "You did not register a node yet.")
	}
	return ctx, nil
}

func (bot *TipBot) nodeHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	splits := strings.Split(m.Text, " ")
	if len(splits) == 1 {
		return bot.getNodeHandler(ctx, m)
	} else if len(splits) > 1 {
		if splits[1] == "invoice" {
			return bot.invHandler(ctx, m)
		}
		if splits[1] == "add" {
			return bot.registerNodeHandler(ctx, m)
		}
		if splits[1] == "check" {
			return bot.satdressCheckInvoiceHandler(ctx, m)
		}
	}
	return ctx, nil
}

func (bot *TipBot) registerNodeHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		return ctx, err
	}

	lndparams, err := parseUserSettingInput(ctx, m)
	if err != nil {
		return ctx, err
	}
	user.Settings.LNDParams = &lndparams
	user.Settings.NodeType = "lnd"
	err = UpdateUserRecord(user, *bot)

	node_info_str := "*Host:*\n`%s`\n*Macaroon:*\n`%s`\n*Cert:*\n`%s`"
	node_info_str_filled := fmt.Sprintf(node_info_str, lndparams.Host, lndparams.Macaroon, lndparams.Cert)
	resp_str := fmt.Sprintf("✅ *Node added.*\n\n%s", node_info_str_filled)
	bot.trySendMessage(m.Sender, resp_str)
	return ctx, nil
}

func (bot *TipBot) invHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		return ctx, err
	}

	var amount int64
	if amount_str, err := getArgumentFromCommand(m.Text, 2); err == nil {
		amount, err = getAmount(amount_str)
	} else {
		// todo -- default amount for testing 1 sat, should actually return error
		amount = 1
		// return ctx, err
	}

	// get invoice from user's node
	getInvoiceParams, err := satdress.GetInvoice(
		satdress.GetInvoiceParams{
			Backend: satdress.LNDParams{
				Cert:     []byte(user.Settings.LNDParams.CertString),
				Host:     user.Settings.LNDParams.Host,
				Macaroon: user.Settings.LNDParams.Macaroon,
			},
			Msatoshi: amount * 1000,
		},
	)
	if err != nil {
		log.Errorln(err.Error())
		return ctx, err
	}
	bot.trySendMessage(m.Sender, fmt.Sprintf("PR: `%s`\n\nHash:`%s`\n\nStatus: `%s`", getInvoiceParams.PR, string(getInvoiceParams.Hash), getInvoiceParams.Status))

	// add the getInvoiceParams to cache to check it later
	bot.Cache.Set(fmt.Sprintf("invoice:%d", user.Telegram.ID), getInvoiceParams, &store.Options{Expiration: 24 * time.Hour})

	return ctx, nil
}

func (bot *TipBot) satdressCheckInvoiceHandler(ctx context.Context, m *tb.Message) (context.Context, error) {

	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	if err != nil {
		return ctx, err
	}

	// get the getInvoiceParams from cache
	getInvoiceParamsInterface, err := bot.Cache.Get(fmt.Sprintf("invoice:%d", user.Telegram.ID))
	if err != nil {
		return ctx, err
	}
	getInvoiceParams := getInvoiceParamsInterface.(satdress.CheckInvoiceParams)
	deadLineCtx, _ := context.WithDeadline(ctx, time.Now().Add(time.Second*30))
	tickerContext, cancel := context.WithCancel(deadLineCtx)
	runtime.NewRetryTicker(tickerContext, "test", runtime.WithRetryDuration(time.Second)).Do(func() {
		// get invoice from user's node
		getInvoiceParams, err = satdress.CheckInvoice(getInvoiceParams)
		if err != nil {
			log.Errorln(err.Error())
			return
		}
		if getInvoiceParams.Status == "SETTLED" {
			cancel()
		}
		bot.trySendMessage(m.Sender, fmt.Sprintf("PR: `%s`\n\nHash:`%s`\n\nStatus: `%s`", getInvoiceParams.PR, string(getInvoiceParams.Hash), getInvoiceParams.Status))
	})

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
