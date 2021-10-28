package telegram

import (
	"context"
	"fmt"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func (bot *TipBot) transactionsHandler(ctx context.Context, m *tb.Message) {
	user := LoadUser(ctx)
	// var payments lnbits.Payments
	payments, _ := bot.Client.Payments(*user.Wallet)
	txstr := ""
	// for _, p := range payments {
	for i := len(payments) - 1; i >= 0; i-- {
		p := payments[i]
		if p.Pending {
			continue
		}

		if p.Amount < 0 {
			txstr += "⬆"
		} else {
			txstr += "⬇"
		}

		timestr := time.Unix(int64(p.Time), 0).UTC().Format("2006-01-02 15:04:05")
		txstr += fmt.Sprintf(" %s:", timestr)
		txstr += fmt.Sprintf(" %+d sat", p.Amount/1000)
		if p.Fee > 0 {
			txstr += fmt.Sprintf("\t(fee: %d sat)", p.Fee/1000)
		}
		memo := p.Memo
		memo_maxlen := 30
		if len(memo) > memo_maxlen {
			memo = memo[:memo_maxlen] + "..."
		}
		if len(memo) > 0 {
			txstr += fmt.Sprintf("\t✉️ %s", memo)
		}
		txstr += "\n"
	}
	txstr = fmt.Sprintf("`%s`", txstr)
	txstr += fmt.Sprintf("\nTotal: %d transactions", len(payments))
	bot.trySendMessage(m.Sender, txstr)
}
