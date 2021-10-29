package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/LightningTipBot/LightningTipBot/internal/lnbits"
	"github.com/eko/gocache/store"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

type TransactionsList struct {
	User         *lnbits.User    `json:"from"`
	Payments     lnbits.Payments `json:"payments"`
	LanguageCode string          `json:"languagecode"`
}

func (txlist *TransactionsList) printTransactions(ctx context.Context, payments lnbits.Payments, pagrenr int) string {
	txstr := ""
	// for _, p := range payments {
	tx_per_page := 10
	pagenr := pagrenr
	if pagenr > (len(payments)+1)/tx_per_page {
		pagenr = 0
	}
	if len(payments) < tx_per_page {
		tx_per_page = len(payments)
	}
	start := pagrenr * (tx_per_page - 1)
	end := start + tx_per_page
	if end > len(payments) {
		end = len(payments)
	}
	for i := len(payments) - 1 - start; i >= len(payments)-1-end; i-- {
		p := payments[i]
		if p.Amount < 0 {
			txstr += "🔼"
		} else {
			txstr += "🔽"
		}
		if p.Pending {
			txstr += "🔄"
		}
		timestr := time.Unix(int64(p.Time), 0).UTC().Format("2006-01-02 15:04:05")
		txstr += fmt.Sprintf(" %s:", timestr)
		txstr += fmt.Sprintf(" %+d sat", p.Amount/1000)
		if p.Fee > 0 {
			txstr += fmt.Sprintf("\t(fee: %d sat)", p.Fee/1000)
		}
		memo := p.Memo
		memo_maxlen := 50
		if len(memo) > memo_maxlen {
			memo = memo[:memo_maxlen] + "..."
		}
		if len(memo) > 0 {
			txstr += fmt.Sprintf("\t✉️ %s", memo)
		}
		txstr += "\n"
	}
	return txstr
}

func (bot *TipBot) transactionsHandler(ctx context.Context, m *tb.Message) {
	user := LoadUser(ctx)
	var payments lnbits.Payments
	paymentsInterface, err := bot.Cache.Get(fmt.Sprintf("%s_transactions", user.Name))
	if err != nil {
		log.Info("Getting from lnbits")
		paymentsInterface, err = bot.Client.Payments(*user.Wallet)
		if err != nil {
			log.Errorf("[transactions] Error: %s", err.Error())
			return
		}
		bot.Cache.Set(fmt.Sprintf("%s_transactions", user.Name), paymentsInterface, &store.Options{Expiration: 1 * time.Minute})
	}
	payments = paymentsInterface.(lnbits.Payments)
	transactionsList := TransactionsList{
		User:         user,
		Payments:     payments,
		LanguageCode: ctx.Value("userLanguageCode").(string),
	}
	txstr := transactionsList.printTransactions(ctx, payments, 0)
	txstr += fmt.Sprintf("\nTotal: %d transactions", len(payments))
	bot.trySendMessage(m.Sender, txstr)
}
