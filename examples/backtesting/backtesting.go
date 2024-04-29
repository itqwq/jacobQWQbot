package main

import (
	"context"

	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/examples/strategies"
	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/plot"
	"github.com/rodrigo-brito/ninjabot/plot/indicator"
	"github.com/rodrigo-brito/ninjabot/storage"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

/*

这段代码展示了如何使用 NinjaBot 进行策略回测。首先，它通过 CSV 文件加载历史数据，初始化一个模拟的交易策略。接着，它设置了一个模拟钱包，并使用该钱包和加载的数据创建一个交易机器人。此机器人配置了可视化图表，展示策略指标和额外的 RSI 指标。最后，该程序运行回测，打印交易结果，并在本地浏览器中显示交易图表。这是一个典型的加密货币交易策略回测流程，涵盖从数据加载到结果展示的全过程。
*/
// 主函数入口
func main() {
	// 创建一个上下文，用于后续可能的取消或超时操作，这是 Go 语言中用于控制多个 goroutine 之间的生命周期（如取消和超时）的标准机制。
	ctx := context.Background()

	// 配置机器人的基本设置，例如交易对
	settings := ninjabot.Settings{
		Pairs: []string{
			"BTCUSDT", // 比特币/美元交易对
			"ETHUSDT", // 以太坊/美元交易对
		},
	}

	// 初始化交易策略，这里使用的是交叉EMA策略
	//要进行这样的回测，我们需要具体的收集到数据和一些编程知识来实现。虽然我无法直接执行代码或访问实时的金融数据库，但我可以提供一个详细的步骤说明，你可以在本地环境中运行这些步骤来测试你的交易策略。
	strategy := new(strategies.CrossEMA)

	// 从CSV文件加载历史数据，为回测提供数据支持
	//这段代码主要是从指定的 CSV 文件（如 "testdata/btc-1h.csv"）中加载特定交易对的历史数据，并通过调用 NewCSVFeed 方法将这些数据以及策略相关的时间框架信息封装到 CSVFeed 结构体中。
	//首先从 CSV 文件中读取历史交易数据，并将其封装到 CSVFeed 结构体中，交易策略需要在代码中明确实现。例如，你可能有一个基于两个不同周期EMA的交叉策略，，创建和配置 NinjaBot 实例，这个机器人将使用前面准备的模拟钱包、策略和数据。给机器人设置策略。将数据源（CSVFeed）连接到机器人。
	csvFeed, err := exchange.NewCSVFeed(
		strategy.Timeframe(), // 使用策略定义的时间框架
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      "testcsv/btc.csv", // 比特币数据文件
			Timeframe: "1h",              // 1小时数据
		},
		exchange.PairFeed{
			Pair:      "ETHUSDT",
			File:      "testcsv/eth.csv", // 以太坊数据文件
			Timeframe: "1h",              // 1小时数据
		},
	)
	if err != nil {
		log.Fatal(err) // 加载数据出错则终止程序
	}

	// 初始化内存数据库，用于存储交易和订单数据
	storage, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err) // 初始化数据库失败则终止程序
	}

	// 创建一个纸上模拟钱包，用于模拟交易过程中的资金流动
	wallet := exchange.NewPaperWallet(
		ctx,
		"USDT",                                  // 使用美元稳定币作为基础资金
		exchange.WithPaperAsset("USDT", 100000), // 初始资金为10,000 USDT, 用于为模拟钱包设置初始资金。在这个例子中，模拟钱包初始拥有 10,000 USDT。
		exchange.WithDataFeed(csvFeed),          // 连接数据源,钱包能够访问和使用历史数据来模拟交易。
	)

	// 创建交易图表，显示策略指标和自定义的RSI指标
	chart, err := plot.NewChart(
		//通过 WithStrategyIndicators 方法将策略相关的指标集成到图表中。这意味着图表将显示策略计算的任何指标，如移动平均线、趋势线等，这取决于策略的具体实现
		//例如：设计了一个策略：当短期移动平均线（比如10天平均线）上穿长期移动平均线（比如50天平均线）时，认为是一个买入信号；当短期线下穿长期线时，认为是一个卖出信号。在图表上，你会看到两条线随着时间推移而变化，交叉点明显标出买入或卖出的信号。 就看到两条线怎么动的
		plot.WithStrategyIndicators(strategy),
		//我为这个图表添加一个周期为14，颜色为紫色RSI指标线，当RSI值超过70时，市场可能处于超买状态，这可能是一个卖出信号；当RSI值低于30时，市场可能处于超卖状态，这可能是一个买入信号。通过观察图表上的紫色RSI线，交易者可以更容易地判断何时进入或退出市场。
		plot.WithCustomIndicators(
			indicator.RSI(14, "purple"), // 添加RSI指标，周期为14，颜色为紫色
		),
		//通过将图表与模拟钱包关联，你可以直接在图表上看到每次交易的执行情况，包括买卖时间点、交易金额和交易结果。图表还将显示模拟钱包的资金变动，帮助你理解每次交易对资金状况的影响。模拟钱包的资金变动, 假设你开始时有10,000美元，在图表上这将表示为起始点,如果一次交易盈利，例如买入比特币后价格上涨并卖出，资金增加到10,500美元，图表上的资金线会上升到10,500美元的位置。如果下一次交易亏损，资金减少到10,200美元，图表上的线会相应下降。
		plot.WithPaperWallet(wallet),
	)
	if err != nil {
		log.Fatal(err) // 创建图表失败则终止程序
	}

	// 初始化 NinjaBot，配置之前创建的组件
	//这段代码的作用是将之前配置好的组件集成到 NinjaBot 实例中，创建一个配置完整的交易机器人，这个机器人将用于执行交易策略、进行回测、处理数据，并生成相关的视觉输出。
	bot, err := ninjabot.NewBot(
		ctx,
		//这包含了交易机器人的基本设置，如交易对和其他重要的配置
		settings,
		//这是之前的模拟钱包，用于模拟交易中的资金流动
		wallet,
		//交易策略
		strategy,
		// 设置为回测模式
		ninjabot.WithBacktest(wallet),
		//存储系统用于记录和持久化所有的交易数据、订单信息以及可能的状态数据。这对于进行事后分析、监控机器人的行为以及确保数据的完整性和可追溯性非常重要。
		ninjabot.WithStorage(storage),
		// 将图表订阅到K线数据，这允许图表实时显示K线数据。K线数据包括开盘价、收盘价、最高价和最低价，是金融分析中常用的数据类型，用于可视化资产价格的时间序列变化。通过这种订阅，交易策略中使用的图表可以直接反映市场的实时或历史价格动态，帮助交易者更好地理解市场趋势和价格行为
		ninjabot.WithCandleSubscription(chart),
		//将图表订阅到订单数据,这个配置确保每当有新的订单被创建或现有订单被更新（如执行、修改或取消）时，相关信息都会在图表上显示。这为交易者提供了一个直观的视图，以监控和评估他们的订单如何在市场中被执行，以及这些订单如何影响他们的交易策略
		ninjabot.WithOrderSubscription(chart),
		//设置日志记录的级别为警告，这个设置调整了日志系统记录信息的详细程度。在警告级别，系统仅记录重要或潜在的问题警告，而不是所有操作的细节。这有助于减少日志文件的大小和复杂性，让运维人员更容易关注和处理可能的问题和异常，而不必从大量的日常日志中筛选信息。
		ninjabot.WithLogLevel(log.WarnLevel),
	)
	if err != nil {
		log.Fatal(err) // 初始化机器人失败则终止程序
	}

	// 运行机器人，开始模拟交易
	//这个方法的作用是启动机器人，让它开始根据之前配置的策略和设置进行交易模拟。这包括监听市场数据变化、执行交易策略、处理交易订单等。
	err = bot.Run(ctx)
	if err != nil {
		log.Fatal(err) // 运行失败则终止程序
	}

	// 打印回测结果
	//这一行调用 NinjaBot 实例的 Summary 方法，该方法负责总结并输出交易机器人的运行结果。这通常包括交易统计数据，如总盈亏、胜率、最大回撤等关键指标。这些数据对于评估交易策略的效果非常有用
	bot.Summary()

	// 在本地浏览器中显示交易图表，这段代码的目的确实是根据之前配置好的 chart 图表，启动并显示图表，以便在本地浏览器中查看包括订单信息、K线数据和其他交易指标在内的所有相关数据。
	err = chart.Start()
	if err != nil {
		log.Fatal(err) // 显示图表失败则终止程序
	}
}
