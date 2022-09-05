package telegram

import (
	"fmt"
	"strings"

	"github.com/LightningTipBot/LightningTipBot/internal/telegram/intercept"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/lightningtipbot/telebot.v3"
)

type InterceptionWrapper struct {
	Endpoints   []interface{}
	Handler     intercept.Func
	Interceptor *Interceptor
}

// registerTelegramHandlers will register all Telegram handlers.
func (bot TipBot) registerTelegramHandlers() {
	telegramHandlerRegistration.Do(func() {
		// Set up handlers
		log.WithFields(log.Fields{
			"module": "telegram",
			"func":   "registerTelegramHandlers"}).Traceln("registering endpoints")
		for _, h := range bot.getHandler() {
			bot.register(h)
		}

	})
}

func getDefaultBeforeInterceptor(bot TipBot) []intercept.Func {
	return []intercept.Func{bot.idInterceptor}
}
func getDefaultDeferInterceptor(bot TipBot) []intercept.Func {
	return []intercept.Func{bot.unlockInterceptor}
}
func getDefaultAfterInterceptor(bot TipBot) []intercept.Func {
	return []intercept.Func{}
}

// registerHandlerWithInterceptor will register a ctx with all the predefined interceptors, based on the interceptor type
func (bot TipBot) registerHandlerWithInterceptor(h InterceptionWrapper) {
	h.Interceptor.Before = append(getDefaultBeforeInterceptor(bot), h.Interceptor.Before...)
	//h.Interceptor.After = append(h.Interceptor.After, getDefaultAfterInterceptor(bot)...)
	//h.Interceptor.OnDefer = append(h.Interceptor.OnDefer, getDefaultDeferInterceptor(bot)...)
	for _, endpoint := range h.Endpoints {
		bot.handle(endpoint, intercept.WithHandler(h.Handler,
			intercept.WithBefore(h.Interceptor.Before...),
			intercept.WithAfter(h.Interceptor.After...),
			intercept.WithDefer(h.Interceptor.OnDefer...),
			intercept.WithDefaultLogFields(h.Interceptor.Fields)))
	}
}

// handle accepts an endpoint and handler for Telegram handler registration.
// function will automatically register string handlers as uppercase and first letter uppercase.
func (bot TipBot) handle(endpoint interface{}, handler tb.HandlerFunc) {
	// register the endpoint
	bot.Telegram.Handle(endpoint, handler)
	switch endpoint.(type) {
	case string:
		// check if this is a string endpoint
		sEndpoint := endpoint.(string)
		if strings.HasPrefix(sEndpoint, "/") {
			// Uppercase endpoint registration, because starting with slash
			bot.Telegram.Handle(strings.ToUpper(sEndpoint), handler)
			if len(sEndpoint) > 2 {
				// Also register endpoint with first letter uppercase
				bot.Telegram.Handle(fmt.Sprintf("/%s%s", strings.ToUpper(string(sEndpoint[1])), sEndpoint[2:]), handler)
			}
		}
	}
}

// register registers a handler, so that Telegram can handle the endpoint correctly.
func (bot TipBot) register(h InterceptionWrapper) {
	if h.Interceptor != nil {
		bot.registerHandlerWithInterceptor(h)
	} else {
		for _, endpoint := range h.Endpoints {
			bot.handle(endpoint, intercept.WithHandler(h.Handler))
		}
	}
}

// getHandler returns a list of all handlers, that need to be registered with Telegram
func (bot TipBot) getHandler() []InterceptionWrapper {
	return []InterceptionWrapper{
		{
			Endpoints: []interface{}{"/start"},
			Handler:   bot.startHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "startHandler",
					"path":   "/start",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/generate"},
			Handler:   bot.generateImages,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "generateImages",
					"path":   "/generate",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/tip", "/t", "/honk"},
			Handler:   bot.tipHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "tipHandler",
					"path":   "/tip",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.loadReplyToInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/pay"},
			Handler:   bot.payHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "payHandler",
					"path":   "/pay",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/invoice", &btnInvoiceMainMenu},
			Handler:   bot.invoiceHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "invoiceHandler",
					"path":   "/invoice",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/node"},
			Handler:   bot.nodeHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "nodeHandler",
					"path":   "/node",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnSatdressCheckInvoice},
			Handler:   bot.satdressCheckInvoiceHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "satdressCheckInvoiceHandler",
					"path":   "btnSatdressCheckInvoice",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/shops"},
			Handler:   bot.shopsHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsHandler",
					"path":   "/shops",
				},
				Before: []intercept.Func{
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.loadUserInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{"/shop"},
			Handler:   bot.shopHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopHandler",
					"path":   "/shop",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.loadUserInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{"/balance", &btnBalanceMainMenu},
			Handler:   bot.balanceHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "balanceHandler",
					"path":   "/balance",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/send", &btnSendMenuEnter},
			Handler:   bot.sendHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "sendHandler",
					"path":   "/send",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.loadReplyToInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnSendMainMenu},
			Handler:   bot.keyboardSendHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "keyboardSendHandler",
					"path":   "btnSendMainMenu",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.loadReplyToInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		// previously, this was the send menu but it
		// was replaced with the webapp
		// {
		// 	Endpoints: []interface{}{&btnWebAppMainMenu},
		// 	Handler:   bot.keyboardSendHandler,
		// 	Interceptor: &Interceptor{

		// 		Before: []intercept.Func{
		// 			bot.localizerInterceptor,
		// 			bot.logMessageInterceptor,
		// 			bot.requireUserInterceptor,
		// 			bot.loadReplyToInterceptor,
		// 			bot.lockInterceptor,
		// 		},
		// 		OnDefer: []intercept.Func{
		// 			bot.unlockInterceptor,
		// 		},
		// 	},
		// },
		{
			Endpoints: []interface{}{"/transactions"},
			Handler:   bot.transactionsHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "transactionsHandler",
					"path":   "/transactions",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&btnLeftTransactionsButton},
			Handler:   bot.transactionsScrollLeftHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "transactionsScrollLeftHandler",
					"path":   "btnLeftTransactionsButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnRightTransactionsButton},
			Handler:   bot.transactionsScrollRightHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "transactionsScrollRightHandler",
					"path":   "btnRightTransactionsButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/faucet", "/zapfhahn", "/kraan", "/grifo"},
			Handler:   bot.faucetHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "faucetHandler",
					"path":   "/faucet",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/tipjar", "/spendendose"},
			Handler:   bot.tipjarHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "tipjarHandler",
					"path":   "/tipjar",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/help", &btnHelpMainMenu},
			Handler:   bot.helpHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "helpHandler",
					"path":   "/help",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/basics"},
			Handler:   bot.basicsHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "basicsHandler",
					"path":   "/basics",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/donate"},
			Handler:   bot.donationHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "donationHandler",
					"path":   "/donate",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/advanced"},
			Handler:   bot.advancedHelpHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "advancedHelpHandler",
					"path":   "/advanced",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/link"},
			Handler:   bot.lndhubHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "lndhubHandler",
					"path":   "/link",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/api"},
			Handler:   bot.apiHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "apiHandler",
					"path":   "/api",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/lnurl"},
			Handler:   bot.lnurlHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "lnurlHandler",
					"path":   "/lnurl",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		// group tickets
		{
			Endpoints: []interface{}{"/group"},
			Handler:   bot.groupHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "groupHandler",
					"path":   "/group",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{"/join"},
			Handler:   bot.groupRequestJoinHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "groupRequestJoinHandler",
					"path":   "/join",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.startUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnPayTicket},
			Handler:   bot.groupConfirmPayButtonHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "groupConfirmPayButtonHandler",
					"path":   "btnPayTicket",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{tb.OnPhoto},
			Handler:   bot.photoHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "photoHandler",
					"path":   "<photo>",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{tb.OnDocument, tb.OnVideo, tb.OnAnimation, tb.OnVoice, tb.OnAudio, tb.OnSticker, tb.OnVideoNote},
			Handler:   bot.fileHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "fileHandler",
					"path":   "<file>",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor,
					bot.loadUserInterceptor,
					bot.logMessageInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{tb.OnText},
			Handler:   bot.anyTextHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "anyTextHandler",
					"path":   "<text>",
				},
				Before: []intercept.Func{
					bot.requirePrivateChatInterceptor, // Respond to any text only in private chat
					bot.localizerInterceptor,
					bot.loadUserInterceptor, // need to use loadUserInterceptor instead of requireUserInterceptor, because user might not be registered yet
					bot.logMessageInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{tb.OnQuery},
			Handler:   bot.anyQueryHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "anyQueryHandler",
					"path":   "<query>",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{tb.OnInlineResult},
			Handler:   bot.anyChosenInlineHandler,
		},
		{
			Endpoints: []interface{}{&btnPay},
			Handler:   bot.confirmPayHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "confirmPayHandler",
					"path":   "btnPay",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelPay},
			Handler:   bot.cancelPaymentHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelPaymentHandler",
					"path":   "btnCancelPay",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnSend},
			Handler:   bot.confirmSendHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "confirmSendHandler",
					"path":   "btnSend",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelSend},
			Handler:   bot.cancelSendHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelSendHandler",
					"path":   "btnCancelSend",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnAcceptInlineSend},
			Handler:   bot.acceptInlineSendHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "acceptInlineSendHandler",
					"path":   "btnAcceptInlineSend",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelInlineSend},
			Handler:   bot.cancelInlineSendHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelInlineSendHandler",
					"path":   "btnCancelInlineSend",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnAcceptInlineReceive},
			Handler:   bot.acceptInlineReceiveHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "acceptInlineReceiveHandler",
					"path":   "btnAcceptInlineReceive",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelInlineReceive},
			Handler:   bot.cancelInlineReceiveHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelInlineReceiveHandler",
					"path":   "btnCancelInlineReceive",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnAcceptInlineFaucet},
			Handler:   bot.acceptInlineFaucetHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "acceptInlineFaucetHandler",
					"path":   "btnAcceptInlineFaucet",
				},
				Before: []intercept.Func{
					bot.singletonCallbackInterceptor,
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
					bot.answerCallbackInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelInlineFaucet},
			Handler:   bot.cancelInlineFaucetHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelInlineFaucetHandler",
					"path":   "btnCancelInlineFaucet",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnAcceptInlineTipjar},
			Handler:   bot.acceptInlineTipjarHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "acceptInlineTipjarHandler",
					"path":   "btnAcceptInlineTipjar",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelInlineTipjar},
			Handler:   bot.cancelInlineTipjarHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelInlineTipjarHandler",
					"path":   "btnCancelInlineTipjar",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnWithdraw},
			Handler:   bot.confirmWithdrawHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "confirmWithdrawHandler",
					"path":   "btnWithdraw",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelWithdraw},
			Handler:   bot.cancelWithdrawHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelWithdrawHandler",
					"path":   "btnCancelWithdraw",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnAuth},
			Handler:   bot.confirmLnurlAuthHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "confirmLnurlAuthHandler",
					"path":   "btnAuth",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&btnCancelAuth},
			Handler:   bot.cancelLnurlAuthHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "cancelLnurlAuthHandler",
					"path":   "btnCancelAuth",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.requireUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				},
			},
		},
		{
			Endpoints: []interface{}{&shopNewShopButton},
			Handler:   bot.shopNewShopHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopNewShopHandler",
					"path":   "shopNewShopButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopAddItemButton},
			Handler:   bot.shopNewItemHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopNewItemHandler",
					"path":   "shopAddItemButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopBuyitemButton},
			Handler:   bot.shopGetItemFilesHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopGetItemFilesHandler",
					"path":   "shopBuyitemButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopNextitemButton},
			Handler:   bot.shopNextItemButtonHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopNextItemButtonHandler",
					"path":   "shopNextitemButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&browseShopButton},
			Handler:   bot.shopsBrowser,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsBrowser",
					"path":   "browseShopButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopSelectButton},
			Handler:   bot.shopSelect,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopSelect",
					"path":   "shopSelectButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that opens selection of shops to delete
		{
			Endpoints: []interface{}{&shopDeleteShopButton},
			Handler:   bot.shopsDeleteShopBrowser,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsDeleteShopBrowser",
					"path":   "shopDeleteShopButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that selects which shop to delete
		{
			Endpoints: []interface{}{&shopDeleteSelectButton},
			Handler:   bot.shopSelectDelete,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopSelectDelete",
					"path":   "shopDeleteSelectButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that opens selection of shops to get links of
		{
			Endpoints: []interface{}{&shopLinkShopButton},
			Handler:   bot.shopsLinkShopBrowser,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsLinkShopBrowser",
					"path":   "shopLinkShopButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that selects which shop to link
		{
			Endpoints: []interface{}{&shopLinkSelectButton},
			Handler:   bot.shopSelectLink,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopSelectLink",
					"path":   "shopLinkSelectButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that opens selection of shops to rename
		{
			Endpoints: []interface{}{&shopRenameShopButton},
			Handler:   bot.shopsRenameShopBrowser,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsRenameShopBrowser",
					"path":   "shopRenameShopButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that selects which shop to rename
		{
			Endpoints: []interface{}{&shopRenameSelectButton},
			Handler:   bot.shopSelectRename,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopSelectRename",
					"path":   "shopRenameSelectButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that opens shops settings buttons view
		{
			Endpoints: []interface{}{&shopSettingsButton},
			Handler:   bot.shopSettingsHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopSettingsHandler",
					"path":   "shopSettingsButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that lets user enter description for shops
		{
			Endpoints: []interface{}{&shopDescriptionShopButton},
			Handler:   bot.shopsDescriptionHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsDescriptionHandler",
					"path":   "shopDescriptionShopButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// button that resets user shops
		{
			Endpoints: []interface{}{&shopResetShopButton},
			Handler:   bot.shopsResetHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsResetHandler",
					"path":   "shopResetShopButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopResetShopAskButton},
			Handler:   bot.shopsAskDeleteAllShopsHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsAskDeleteAllShopsHandler",
					"path":   "shopResetShopAskButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopPrevitemButton},
			Handler:   bot.shopPrevItemButtonHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopPrevItemButtonHandler",
					"path":   "shopPrevitemButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopShopsButton},
			Handler:   bot.shopsHandlerCallback,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopsHandlerCallback",
					"path":   "shopShopsButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		// shop item settings buttons
		{
			Endpoints: []interface{}{&shopItemSettingsButton},
			Handler:   bot.shopItemSettingsHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopItemSettingsHandler",
					"path":   "shopItemSettingsButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopItemSettingsBackButton},
			Handler:   bot.displayShopItemHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "displayShopItemHandler",
					"path":   "shopItemSettingsBackButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopItemDeleteButton},
			Handler:   bot.shopItemDeleteHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopItemDeleteHandler",
					"path":   "shopItemDeleteButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopItemPriceButton},
			Handler:   bot.shopItemPriceHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopItemPriceHandler",
					"path":   "shopItemPriceButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopItemTitleButton},
			Handler:   bot.shopItemTitleHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopItemTitleHandler",
					"path":   "shopItemTitleButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopItemAddFileButton},
			Handler:   bot.shopItemAddItemHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopItemAddItemHandler",
					"path":   "shopItemAddFileButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopItemBuyButton},
			Handler:   bot.shopConfirmBuyHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "shopConfirmBuyHandler",
					"path":   "shopItemBuyButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
		{
			Endpoints: []interface{}{&shopItemCancelBuyButton},
			Handler:   bot.displayShopItemHandler,
			Interceptor: &Interceptor{
				Fields: log.Fields{
					"module": "telegram",
					"func":   "displayShopItemHandler",
					"path":   "shopItemCancelBuyButton",
				},
				Before: []intercept.Func{
					bot.localizerInterceptor,
					bot.loadUserInterceptor,
					bot.answerCallbackInterceptor,
					bot.lockInterceptor,
				},
				OnDefer: []intercept.Func{
					bot.unlockInterceptor,
				}},
		},
	}
}
