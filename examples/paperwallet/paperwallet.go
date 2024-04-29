package main

import (
	"context"
	"os"
	"strconv"

	"github.com/rodrigo-brito/ninjabot/plot"
	"github.com/rodrigo-brito/ninjabot/plot/indicator"

	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/examples/strategies"
	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/storage"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

/*
这段代码实现了一个使用 NinjaBot 库的交易机器人示例，可以在真实或模拟交易环境中运行。它配置了交易对、Telegram 通知、交易策略、数据存储和图表显示等功能，并根据用户的环境变量设置连接到真实或模拟的交易所，并执行相应的交易操作。
*/
// 这个示例展示了如何使用 NinjaBot 在模拟交易中与一个虚拟交易所进行交互。
// paperwallet 是一个未连接到任何交易所的钱包，它是一个模拟交易，具有实时数据。
func main() {
	var (
		ctx             = context.Background()                     // 创建一个背景上下文，通常用于初始化操作
		telegramToken   = os.Getenv("TELEGRAM_TOKEN")              // 从环境变量中获取 Telegram 机器人的令牌
		telegramUser, _ = strconv.Atoi(os.Getenv("TELEGRAM_USER")) // 将环境变量中的 Telegram 用户 ID 转换为整数
	)

	// 配置 NinjaBot 的设置
	settings := ninjabot.Settings{
		Pairs: []string{ // 设置交易对
			"BTCUSDT",
			"ETHUSDT",
			"BNBUSDT",
			"LTCUSDT",
		},
		Telegram: ninjabot.TelegramSettings{ // 配置 Telegram 通知
			// 检查是否启用了 Telegram 通知如果Telegram令牌不为空，用户不为空，就返回true，那么就启动
			Enabled: telegramToken != "" && telegramUser != 0,
			// 设置 Telegram 令牌
			Token: telegramToken,
			// 设置接收通知的用户列表
			Users: []int{telegramUser},
		},
	}

	// 使用 Binance 提供实时数据源，这段代码通过调用 exchange.NewBinance(ctx) 创建了一个与Binance交易所的连接，并返回一个包含了实时数据的Binance实例。ctx参数是上下文，用于控制和管理与Binance交易所的连接。
	binance, err := exchange.NewBinance(ctx)
	if err != nil {
		//遇到致命错误终止获取交易所实时数据
		log.Fatal(err)
	}

	// 创建一个存储以保存交易记录
	storage, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err)
	}

	// 创建一个 paperwallet 来模拟交易所的钱包，进行虚拟操作
	paperWallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		exchange.WithPaperFee(0.001, 0.001),    // 设置手续费
		exchange.WithPaperAsset("USDT", 10000), // 设置模拟钱包中的资产
		//使用 Binance 提供的实时数据，目的是确保 paperwallet 能够获取到与真实交易所相同的实时数据，以便在模拟交易中进行操作。这确保了模拟交易的结果与实际交易所的实时数据对应，意思就是交易所数据下跌，我模拟数据的交易也会亏钱
		exchange.WithDataFeed(binance),
	)

	// 初始化交易策略，CrossEMA 的交叉指数移动平均线策略。
	strategy := new(strategies.CrossEMA)

	// 创建一个图表来展示交易数据
	chart, err := plot.NewChart(
		plot.WithCustomIndicators( // 添加自定义指标
			indicator.EMA(8, "red"),   // 红色的 8 日指数移动平均线
			indicator.SMA(21, "blue"), // 蓝色的 21 日简单移动平均线
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 初始化 NinjaBot
	bot, err := ninjabot.NewBot(
		ctx,
		// NinjaBot 的设置，包括交易对和 Telegram 通知的配置。
		settings,
		//模拟交易所的虚拟钱包，用于进行模拟交易操作。
		paperWallet,
		// 交易策略，用于确定何时进行买卖操作。
		strategy,
		//配置数据存储
		ninjabot.WithStorage(storage),
		// 将 paperwallet 关联到 NinjaBot,通过将钱包与NinjaBot关联，可以让NinjaBot在执行交易操作时直接操作这个钱包，而不必在实际的交易所执行操作。就是直接使用模拟钱包操作
		ninjabot.WithPaperWallet(paperWallet),
		//：将图表订阅到蜡烛图数据，以便实时更新图表展示的交易数据。
		ninjabot.WithCandleSubscription(chart),
		//将图表订阅到订单数据，以便实时更新图表展示的订单信息。
		ninjabot.WithOrderSubscription(chart),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 启动图表服务
	go func() {
		err := chart.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// 运行交易机器人
	err = bot.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
