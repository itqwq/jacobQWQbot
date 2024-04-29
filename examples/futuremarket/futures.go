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
这段 Go 语言代码利用 NinjaBot 库实现了一个自动化的加密货币交易机器人，该机器人配置了通过 Binance 期货市场交易的功能和 Telegram 通知服务。它主要使用环境变量读取敏感信息，设置交易对，并通过交叉 EMA 策略来决定交易时机，旨在自动执行买卖操作以寻求盈利。
*/
// This example shows how to use futures market with NinjaBot.
// main 函数是程序的入口点
func main() {

	//在window以下命令并按回车$env:API_KEY="<your-token-here>"真实token（替换 <your-token-here> 为你的实际 token），为了让交易机器人能够运行，你首先需要在你的环境变量中添加那些它需要的如 API 密钥（token）的值。你的 Go 代码中使用 os.Getenv("API_KEY") 来获取 API_KEY 的值，这意味着在运行交易机器人之前，API_KEY 环境变量必须包含正确的 token 值。这样程序才能正确地进行 API 调用。
	var (
		// 创建一个背景上下文，通常用于初始化操作
		ctx = context.Background()
		//Getenv 是 os 包中的一个函数，它的作用是读取环境变量的值。Getenv 需要一个字符串参数，这个参数是你想要获取的环境变量的名称
		//apiKey 是一个变量，用来存储从环境变量 API_KEY 中获取的值。API_KEY 通常是访问外部 API（如交易所的 API）时需要的一个密钥，它用于身份验证和授权。
		apiKey = os.Getenv("API_KEY")
		// 查找名为 API_SECRET 的环境变量，并返回它的secret值。这个值被赋给了 secretKey 变量，之后你的程序就可以使用这个 secretKey 来进行需要这个密钥的操作，比如与外部服务的安全通信,
		secretKey = os.Getenv("API_SECRET")
		// 从环境变量中获取 Telegram 机器人的令牌
		telegramToken = os.Getenv("TELEGRAM_TOKEN")
		// 将环境变量中的 Telegram 用户 ID 转换为整数
		telegramUser, _ = strconv.Atoi(os.Getenv("TELEGRAM_USER"))
	)

	// 配置 NinjaBot 的设置,交易机器人只会处理两个交易对："BTCUSDT" 和 "ETHUSDT"，分别表示比特币兑美元和以太坊兑美元的交易对。因此，机器人将会在这两个期货市场上执行交易操作，并监控这些交易对的价格变动，以便做出相应的买卖决策。
	settings := ninjabot.Settings{
		Pairs: []string{
			"BTCUSDT", // 交易对 BTC/USDT
			"ETHUSDT", // 交易对 ETH/USDT
		},
		Telegram: ninjabot.TelegramSettings{
			Enabled: true,                // 启用 Telegram 通知
			Token:   telegramToken,       // 设置 Telegram 令牌
			Users:   []int{telegramUser}, // 设置接收通知的用户列表
		},
	}

	// 使用指定的 API 密钥和杠杆设置初始化 Binance 期货交易所实例，以便后续的交易操作能够使用这个交易所进行。
	//隔离保证金 意思就是亏得话只会亏下跌得那个交易对，其他得不妨碍，比如因为BTC价格下跌，你的BTC/USDT头寸亏损了10%。由于使用了隔离保证金，这个亏损只会影响你BTC/USDT头寸的保证金余额，而不会影响到ETH/USDT头寸
	binance, err := exchange.NewBinanceFuture(ctx,
		exchange.WithBinanceFutureCredentials(apiKey, secretKey),                      // 使用 API 密钥和密钥
		exchange.WithBinanceFutureLeverage("BTCUSDT", 5, exchange.MarginTypeIsolated), // BTCUSDT 使用 1 倍杠杆，隔离保证金
		exchange.WithBinanceFutureLeverage("ETHUSDT", 5, exchange.MarginTypeIsolated), // ETHUSDT 同样使用 1 倍杠杆，隔离保证金
	)
	if err != nil {
		log.Fatal(err) // 如果初始化交易所失败，记录致命错误并退出
	}

	// 初始化交易策略和机器人，使用了之前配置好的设置、Binance 期货交易所实例和交易策略。如果在创建过程中出现了错误，它会记录致命错误并退出程序，以防止继续执行后续的操作。
	strategy := new(strategies.CrossEMA)                          // 使用交叉 EMA 策略
	bot, err := ninjabot.NewBot(ctx, settings, binance, strategy) // 创建 NinjaBot 实例
	if err != nil {
		log.Fatalln(err) // 如果创建机器人失败，记录致命错误并退出
	}

	// 运行机器人，运行机器人之后，机器人就会通过我们之前设置好的配置自动进行期货交易
	err = bot.Run(ctx) // 启动机器人，开始交易
	if err != nil {
		log.Fatalln(err) // 如果运行时出错，记录致命错误并退出
	}
}
