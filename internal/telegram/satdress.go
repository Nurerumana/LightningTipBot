package telegram

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/LightningTipBot/LightningTipBot/internal/satdress"
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
	return satdress.LNDParams{
		Cert:     parseCertificateToPem(cert),
		Host:     host,
		Macaroon: macaroon,
	}, nil
}

func (bot *TipBot) nodeHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	splits := strings.Split(m.Text, " ")
	if len(splits) > 1 {
		if splits[1] == "invoice" {
			return bot.invHandler(ctx, m)
		}
		if splits[1] == "add" {
			return bot.registerNodeHandler(ctx, m)
		}
	}
	return ctx, nil
}

func (bot *TipBot) registerNodeHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	user, err := GetLnbitsUserWithSettings(m.Sender, *bot)
	node_info_str := "*Host:*\n`%s`\n*Macaroon:*\n`%s`\n*Cert:*\n`%s`"
	if err != nil {
		log.Infof("Could not get user settings for user %s", GetUserStr(user.Telegram))
		// return ctx, err
	} else {
		if user.Settings.LNDParams.Host != "" {
			node_info_str_filled := fmt.Sprintf(node_info_str, user.Settings.LNDParams.Host, user.Settings.LNDParams.Macaroon, user.Settings.LNDParams.Cert)
			resp_str := fmt.Sprintf("ℹ️ *Your node information.*\n\n%s", node_info_str_filled)
			bot.trySendMessage(m.Sender, resp_str)
		}
	}

	lndparams, err := parseUserSettingInput(ctx, m)

	node_info_str_filled := fmt.Sprintf(node_info_str, lndparams.Host, lndparams.Macaroon, lndparams.Cert)
	resp_str := fmt.Sprintf("✅ *Node added.*\n\n%s", node_info_str_filled)
	bot.trySendMessage(m.Sender, resp_str)
	return ctx, nil
}

func (bot *TipBot) invHandler(ctx context.Context, m *tb.Message) (context.Context, error) {
	// check and print all commands
	// bot.anyTextHandler(ctx, m)
	// user := LoadUser(ctx)

	var amount int64
	if amount_str, err := getArgumentFromCommand(m.Text, 2); err == nil {
		amount, err = getAmount(amount_str)
	} else {
		// todo -- default amount for testing 1 sat, should actually return error
		amount = 1
		// return ctx, err
	}

	// get invoice from user's node
	pr, err := satdress.GetInvoice(
		satdress.Params{
			Backend: satdress.LNDParams{
				Cert:     parseCertificateToPem("2d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d494943496a4343416369674177494241674952414f68774131694b486b4d704b3654734a797836794b4177436759494b6f5a497a6a3045417749774d4445660a4d4230474131554543684d576247356b494746316447396e5a57356c636d46305a575167593256796444454e4d4173474131554541784d455a4746325a5441650a467730794d5445794d4463794d6a51314d544e61467730794d7a41794d4445794d6a51314d544e614d444178487a416442674e5642416f54466d78755a4342680a645852765a3256755a584a686447566b49474e6c636e51784454414c42674e5642414d5442475268646d55775754415442676371686b6a4f50514942426767710a686b6a4f50514d4242774e4341415350514146462f586838655666496d43414f7a6a456d57596d2f736470632b616a535a50654245333342305369787a3433350a30427976344e317033396d54527a4f783848647332777562326e6d505958636d2b6263506f3448434d49472f4d41344741315564447745422f775145417749430a7044415442674e56485355454444414b4267677242674546425163444154415042674e5648524d4241663845425441444151482f4d42304741315564446751570a424254734732594c32666a6744467954366c2b474f777671762b38634d44426f42674e5648524545595442666767526b59585a6c67676c7362324e68624768760a6333534342475268646d574344584276624746794c5734794c575268646d57434248567561586943436e56756158687759574e725a58534342324a315a6d4e760a626d36484248384141414748454141414141414141414141414141414141414141414748424b775741415977436759494b6f5a497a6a304541774944534141770a52514968414f6437436c716e4a3258735571716b5953756e4937777147736e4b596d57334668353045765877775a44394169424e4c6e575a4342416a343664780a726f5a392f435563595a78754756432f6d666b2b38325a2b5073777977413d3d0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a"),
				Host:     "https://127.0.0.1:8085",
				Macaroon: "AgEDbG5kAlgDChDzBenLUGm5pV4AogEZbcHTEgEwGhYKB2FkZHJlc3MSBHJlYWQSBXdyaXRlGhcKCGludm9pY2VzEgRyZWFkEgV3cml0ZRoPCgdvbmNoYWluEgRyZWFkAAAGIFoUGihWFYLwzjJLaqQmTdrNoDcNbb4piZzXi72XlKTS",
			},
			Msatoshi: amount * 1000,
		},
	)
	if err != nil {
		log.Errorln(err.Error())
		return ctx, err
	}

	bot.trySendMessage(m.Sender, pr)

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
