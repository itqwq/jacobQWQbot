package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/examples/strategies"
	"github.com/rodrigo-brito/ninjabot/exchange"
)

/*

这段代码的总体交易逻辑是：从环境变量中获取必要的敏感信息，设置交易对和通知选项，通过 Binance API 初始化交易所实例，然后使用交叉 EMA 策略初始化 NinjaBot 实例。接着，NinjaBot 开始执行现货市场交易策略，并通过 Telegram 进行通知。

通过设置API密钥、密钥和Telegram令牌，配置了NinjaBot的基本设置，包括要交易的货币对和是否启用Telegram通知。然后，使用Binance交易所的API密钥和密钥创建了一个Binance交易所实例。接下来，初始化了一个交易策略（在这里是交叉EMA策略）和NinjaBot实例。最后，运行NinjaBot，它将开始根据给定的交易策略和设置在Binance现货市场上执行交易操作。
*/
// 这个示例演示了如何在Binance中使用NinjaBot进行现货交易
func main() {
	// 创建一个背景上下文
	ctx := context.Background()
	// 从环境变量中获取必要的API密钥、密钥、Telegram令牌和Telegram用户ID
	apiKey := os.Getenv("API_KEY")
	secretKey := os.Getenv("API_SECRET")
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	telegramUser, _ := strconv.Atoi(os.Getenv("TELEGRAM_USER"))
	log.Println(apiKey)
	// 配置NinjaBot的设置，包括交易对和Telegram通知选项
	settings := ninjabot.Settings{
		Pairs: []string{
			"BTCUSDT",
			"ETHUSDT",
		},
		Telegram: ninjabot.TelegramSettings{
			Enabled: true,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}

	// 初始化Binance交易所实例
	binance, err := exchange.NewBinance(ctx, exchange.WithBinanceCredentials(apiKey, secretKey))
	if err != nil {
		log.Fatalln(err)
	}

	// 初始化交易策略和NinjaBot实例
	strategy := new(strategies.CrossEMA)
	bot, err := ninjabot.NewBot(ctx, settings, binance, strategy)
	if err != nil {
		log.Fatalln(err)
	}

	// 运行NinjaBot
	err = bot.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
