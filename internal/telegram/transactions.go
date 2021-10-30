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
	ID           string          `json:"id"`
	User         *lnbits.User    `json:"from"`
	Payments     lnbits.Payments `json:"payments"`
	LanguageCode string          `json:"languagecode"`
	CurrentPage  int             `json:"currentpage"`
	MaxPages     int             `json:"maxpages"`
	TxPerPage    int             `json:"txperpage"`
}

func (txlist *TransactionsList) printTransactions(ctx context.Context) string {
	txstr := ""
	// for _, p := range payments {
	payments := txlist.Payments
	pagenr := txlist.CurrentPage
	tx_per_page := txlist.TxPerPage
	if pagenr > (len(payments)+1)/tx_per_page {
		pagenr = 0
	}
	if len(payments) < tx_per_page {
		tx_per_page = len(payments)
	}
	start := pagenr * (tx_per_page - 1)
	end := start + tx_per_page
	if end >= len(payments) {
		end = len(payments) - 1
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
	txstr += fmt.Sprintf("\nTotal: %d transactions. Page %d of %d.", len(payments), txlist.CurrentPage+1, txlist.MaxPages)
	return txstr
}

var (
	transactionsMeno           = &tb.ReplyMarkup{ResizeReplyKeyboard: true}
	btnLeftTransactionsButton  = inlineTipjarMenu.Data("◀️", "left_transactions")
	btnRightTransactionsButton = inlineTipjarMenu.Data("▶️", "right_transactions")
)

func (bot TipBot) makeTransactionsKeyboard(ctx context.Context, txlist TransactionsList) *tb.ReplyMarkup {
	// transactionsMeno := &tb.ReplyMarkup{ResizeReplyKeyboard: true}
	leftTransactionsButton := transactionsMeno.Data("←", "left_transactions")
	rightTransactionsButton := transactionsMeno.Data("→", "right_transactions")
	leftTransactionsButton.Data = txlist.ID
	rightTransactionsButton.Data = txlist.ID
	transactionsMeno.Inline(
		transactionsMeno.Row(
			leftTransactionsButton,
			rightTransactionsButton),
	)
	return transactionsMeno
}

func (bot *TipBot) transactionsHandler(ctx context.Context, m *tb.Message) {
	user := LoadUser(ctx)
	transactionsListInterface, err := bot.Cache.Get(fmt.Sprintf("%s_transactions", user.Name))
	if err != nil {
		payments, err := bot.Client.Payments(*user.Wallet)
		if err != nil {
			log.Errorf("[transactions] Error: %s", err.Error())
			return
		}
		// var payments lnbits.Payments
		// paymentsInterface, err := bot.Cache.Get(fmt.Sprintf("%s_transactions", user.Name))
		// if err != nil {
		// 	log.Info("Getting from lnbits")
		// 	paymentsInterface, err = bot.Client.Payments(*user.Wallet)
		// 	if err != nil {
		// 		log.Errorf("[transactions] Error: %s", err.Error())
		// 		return
		// 	}
		// 	bot.Cache.Set(fmt.Sprintf("%s_transactions", user.Name), paymentsInterface, &store.Options{Expiration: 1 * time.Minute})
		// }
		// payments = paymentsInterface.(lnbits.Payments)
		tx_per_page := 20
		transactionsListInterface = TransactionsList{
			ID:           fmt.Sprintf("txlist-%d-%s", user.Telegram.ID, RandStringRunes(5)),
			User:         user,
			Payments:     payments,
			LanguageCode: ctx.Value("userLanguageCode").(string),
			CurrentPage:  0,
			TxPerPage:    tx_per_page,
			MaxPages:     (len(payments)+1)/tx_per_page + 1,
		}
	}
	transactionsList := transactionsListInterface.(TransactionsList)
	bot.Cache.Set(fmt.Sprintf("%s_transactions", user.Name), transactionsList, &store.Options{Expiration: 1 * time.Minute})
	txstr := transactionsList.printTransactions(ctx)
	bot.trySendMessage(m.Sender, txstr, bot.makeTransactionsKeyboard(ctx, transactionsList))
}

func (bot *TipBot) transactionsScrollLeftHandler(ctx context.Context, c *tb.Callback) {
	user := LoadUser(ctx)
	transactionsListInterface, err := bot.Cache.Get(fmt.Sprintf("%s_transactions", user.Name))
	if err != nil {
		log.Info("Transactions not in cache anymore")
		return
	}
	transactionsList := transactionsListInterface.(TransactionsList)

	if c.Sender.ID == transactionsList.User.Telegram.ID {
		if transactionsList.CurrentPage < transactionsList.MaxPages-1 {
			transactionsList.CurrentPage++
		} else {
			return
		}
		bot.Cache.Set(fmt.Sprintf("%s_transactions", user.Name), transactionsList, &store.Options{Expiration: 1 * time.Minute})
		bot.tryEditMessage(c.Message, transactionsList.printTransactions(ctx), bot.makeTransactionsKeyboard(ctx, transactionsList))
	}
	return
}

func (bot *TipBot) transactionsScrollRightHandler(ctx context.Context, c *tb.Callback) {
	user := LoadUser(ctx)
	transactionsListInterface, err := bot.Cache.Get(fmt.Sprintf("%s_transactions", user.Name))
	if err != nil {
		log.Info("Transactions not in cache anymore")
		return
	}
	transactionsList := transactionsListInterface.(TransactionsList)

	if c.Sender.ID == transactionsList.User.Telegram.ID {
		if transactionsList.CurrentPage > 0 {
			transactionsList.CurrentPage--
		} else {
			return
		}
		bot.Cache.Set(fmt.Sprintf("%s_transactions", user.Name), transactionsList, &store.Options{Expiration: 1 * time.Minute})
		bot.tryEditMessage(c.Message, transactionsList.printTransactions(ctx), bot.makeTransactionsKeyboard(ctx, transactionsList))
	}
	return
}
