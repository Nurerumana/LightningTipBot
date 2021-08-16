package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

const (
	helpMessage = "ℹ️ *Info*\n_This bot sends Bitcoin tips on the Lightning Network⚡️. The basic unit of tips are Satoshis (sat). 100,000,000 sat = 1 Bitcoin. There will only ever be 21 Million Bitcoin._\n\n" +
		"❤️ *Donate*\n" +
		"_This bot charges no fees but generates costs. If you like to support this bot, please consider a donation. To donate, just tip @LightningTipBot or try_ `/send 1000 @LightningTipBot`\n\n" +
		"⚙️ *Commands*\n" +
		"*/tip* 🏅 Reply to a message to tip it: `/tip <amount> [<memo>]`\n" +
		"*/balance* 👑 Check your balance: `/balance`\n" +
		"*/send* 💸 Send funds to a user: `/send <amount> <@username> [<memo>]`\n" +
		"*/invoice* ⚡️ Receive over Lightning: `/invoice <amount> [<memo>]`\n" +
		"*/pay* ⚡️ Pay over Lightning: `/pay <invoice>`\n" +
		"*/help* 📖 Read this help (more: */info*).\n"

	infoMessage = "🧡 *Bitcoin*\n" +
		"_Bitcoin is the currency of the internet. It is permissionless and decentralized and has no masters and no controling authority. Bitcoin is sound money that is faster, more secure, and more inclusive than the legacy financial system._\n\n" +
		"🧮 *Economnics*\n" +
		"_The smallest unit of Bitcoin are Satoshis (sat) and 100,000,000 sat = 1 Bitcoin. There will only ever be 21 Million Bitcoin. The fiat currency value of Bitcoin can change daily. However, if you live on a Bitcoin standard 1 sat will always equal 1 sat._\n\n" +
		"⚡️ *The Lightning Network*\n" +
		"_The Lightning Network is a payment protocol that enables fast and cheap Bitcoin payments that require almost no energy. It is what scales Bitcoin to the billions of people around the world._\n\n" +
		"📲 *Lightning Wallets*\n" +
		"_Your funds on this bot can be sent to any other Lightning wallet and vice versa. Just create an invoice in your Lightning wallet and send it to the bot. If you don't have a Lightning wallet yet, check out_ [Phoenix](https://phoenix.acinq.co/)_,_ [Breez](https://breez.technology/)_,_ [Muun](https://muun.com/)_ (non-custodial), or_ [Wallet of Satoshi](https://www.walletofsatoshi.com/) _(easy)_.\n\n" +
		"📄 *Open Source*\n" +
		"_This bot is free and_ [open source](https://github.com/LightningTipBot/LightningTipBot) _software. You can run it on your own computer and use it in your own community. That way you don't have to trust anyone to keep your funds safe._\n\n" +
		"❤️ *Donate*\n" +
		"_This bot charges no fees but generates costs. If you like to support this bot, please consider a donation. To donate, just tip @LightningTipBot or try_ `/send 1000 @LightningTipBot`"
)

// func helpHowtoUse() string {
// 	return "ℹ️ *Info*\n_This bot sends Bitcoin tips on the Lightning Network⚡️. The basic unit of tips are Satoshis (sat). 100,000,000 sat = 1 Bitcoin. There will only ever be 21 Million Bitcoin._\n\n" +
// 		"❤️ *Donate*\n" +
// 		"_This bot charges no fees but generates costs. If you like to support this bot, please consider a donation. To donate, just tip @LightningTipBot or try_ `/send 1000 @LightningTipBot`\n\n" +
// 		"⚙️ *Commands*\n" +
// 		"*/tip* 🏅 Reply to a message to tip it: `/tip <amount> [<memo>]`\n" +
// 		"*/balance* 👑 Check your balance: `/balance`\n" +
// 		"*/send* 💸 Send funds to a user: `/send <amount> <@username> [<memo>]`\n" +
// 		"*/invoice* ⚡️ Receive over Lightning: `/invoice <amount> [<memo>]`\n" +
// 		"*/pay* ⚡️ Pay over Lightning: `/pay <invoice>`\n" +
// 		"*/help* 📖 Read this help (more: */info*).\n"
// }

func (bot TipBot) helpHandler(m *tb.Message) {
	if !m.Private() {
		// delete message
		NewMessage(m).Dispose(0, bot.telegram)
	}
	bot.telegram.Send(m.Sender, helpMessage, tb.NoPreview)
	return
}

func (bot TipBot) infoHandler(m *tb.Message) {
	if !m.Private() {
		// delete message
		NewMessage(m).Dispose(0, bot.telegram)
	}
	bot.telegram.Send(m.Sender, infoMessage, tb.NoPreview)
	return
}
