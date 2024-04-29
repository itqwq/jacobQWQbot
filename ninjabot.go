package ninjabot // 声明包名为ninjabot

// 导入所需的包和依赖
import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aybabtme/uniplot/histogram"

	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/notification"
	"github.com/rodrigo-brito/ninjabot/order"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/storage"
	"github.com/rodrigo-brito/ninjabot/strategy"
	"github.com/rodrigo-brito/ninjabot/tools/log"
	"github.com/rodrigo-brito/ninjabot/tools/metrics"

	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
)

// 这段代码中，const defaultDatabase = "ninjabot.db"只是定义了一个默认的数据库文件名。它并没有指明具体的文件路径或创建文件。如果指定的文件（在这个例子中是ninjabot.db）不存在，数据库管理系统（如SQLite）会在指定位置创建这个文件。
const defaultDatabase = "ninjabot.db" // 定义默认数据库文件名

func init() {
	// 初始化日志格式设置
	//调用log包的SetFormatter方法来设置日志消息的格式
	//创建了一个TextFormatter的实例，它是log包中定义的一个结构体。这个结构体用于定义文本格式日志的格式。
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,               //这个字段设置为true表示在日志中完整显示时间戳,完整显示时间戳”意味着在每条日志信息中都会包含日期和时间信息，让你能够知道每个日志条目是什么时候产生的。时间戳通常包括年、月、日、小时、分钟，有时还包括秒和毫秒，这取决于时间格式的具体设置。
		TimestampFormat: "2006-01-02 15:04", //这里指定的格式意味着时间戳会以年-月-日 时:分的形式展示，没有秒。
	})
}

// OrderSubscriber 接口，任何实现了OnOrder方法的类型都可以订阅订单更新
type OrderSubscriber interface {
	OnOrder(model.Order)
}

// CandleSubscriber 接口，任何实现了OnCandle方法的类型都可以订阅K线(蜡烛图)数据更新
type CandleSubscriber interface {
	OnCandle(model.Candle)
}

// NinjaBot 结构体定义了机器人的主体结构
type NinjaBot struct {
	// 这是一个存储接口，用于数据的持久化。这可能包括保存交易数据、用户设置或其他重要信息。
	storage storage.Storage
	// 这个字段包含机器人的配置设置，比如交易对、交易参数和可能的运行选项等。
	settings model.Settings
	//交易所接口，允许机器人与具体的交易所进行交互，执行如买卖等操作。
	exchange service.Exchange
	//这是定义机器人交易逻辑的策略接口。它决定了如何根据市场数据来做出交易决策。
	strategy strategy.Strategy
	//通知服务接口，用于发送交易或其他重要事件的通知。
	notifier service.Notifier
	//通过Telegram服务发送通知的接口。这可以用来直接通过Telegram向用户报告交易状态或警告。
	telegram service.Telegram

	//订单控制器，负责管理订单的生命周期，包括订单的创建、修改和取消等。
	orderController *order.Controller
	//用于管理K线（蜡烛图）数据的优先队列，负责按照某种优先级顺序处理数据流。
	priorityQueueCandle   *model.PriorityQueue            // K线数据的优先队列，用于管理数据流
	strategiesControllers map[string]*strategy.Controller // 策略控制器集合，每个交易对一个策略控制器
	orderFeed             *order.Feed                     // 订单更新订阅源
	dataFeed              *exchange.DataFeedSubscription  // 数据订阅源，订阅交易所的数据流
	paperWallet           *exchange.PaperWallet           // 模拟钱包，用于回测和模拟交易

	backtest bool // 一个标志，指示机器人是否处于回测模式。在回测模式下，机器人不会执行实际的交易命令，而是通过历史数据来测试策略的表现。
}

// 函数选项模式（Functional Options Pattern）。这个模式允许你通过函数来设置对象的配置选项，使得对象构造过程更加灵活，并且可以很容易地扩展新的选项而不影响现有代码。
type Option func(*NinjaBot)

// NewBot 创建一个新的 NinjaBot 实例，配置其基本组件和可选设置。
// ctx：上下文对象，用于管理异步任务和超时。
// settings：提供了机器人的配置设置，如交易对和其他运行参数。
// exch：交易所接口，机器人通过此接口与交易所进行交互。
// str：定义机器人的交易策略。
// options：一个或多个配置函数，用于定制机器人的额外行为或设置。
func NewBot(ctx context.Context, settings model.Settings, exch service.Exchange, str strategy.Strategy,
	options ...Option) (*NinjaBot, error) {

	// 初始化 NinjaBot 实例的基本属性。
	bot := &NinjaBot{
		// 将机器人的设置赋值给 bot 的 settings 字段，这些设置包括交易对、API密钥等配置。
		settings: settings,
		// 将传入的交易所接口对象赋值给 bot 的 exchange 字段，使得机器人能够与具体的交易所进行交互。
		exchange: exch,
		// 将传入的交易策略对象赋值给 bot 的 strategy 字段，定义机器人如何根据市场数据做出买卖决策。
		strategy: str,
		// 创建一个新的订单订阅源实例，赋值给 bot 的 orderFeed 字段，用于管理订单的生命周期事件。
		orderFeed: order.NewOrderFeed(),
		//这行代码确实是用来创建一个新的数据订阅源实例，这个实例使用了传入的交易所接口 exch。这个数据订阅源主要用途是从交易所接收实时市场数据，包括但不限于蜡烛图（K线图）数据。
		dataFeed: exchange.NewDataFeed(exch),
		// 初始化一个空的映射，赋值给 bot 的 strategiesControllers 字段，用于存储不同交易对的策略控制器。
		strategiesControllers: make(map[string]*strategy.Controller),
		// 创建一个新的优先队列实例，赋值给 bot 的 priorityQueueCandle 字段，用于管理和排序接收到的K线数据。
		priorityQueueCandle: model.NewPriorityQueue(nil),
	}

	// 验证 settings 中的交易对是否有效。
	for _, pair := range settings.Pairs {
		asset, quote := exchange.SplitAssetQuote(pair)
		if asset == "" || quote == "" {
			return nil, fmt.Errorf("invalid pair: %s", pair)
		}
	}

	// 应用所有提供的配置选项到机器人实例。
	for _, option := range options {
		option(bot)
	}

	var err error
	//检查是否已经有一个存储接口被设置到 NinjaBot 实例中。如果 storage 字段是 nil，意味着还没有任何存储方式被指定。
	// 检查 bot.storage 是否已经初始化。如果 bot.storage 是 nil，表示尚未设置任何存储接口。
	if bot.storage == nil {
		// 如果未指定存储方式，则使用默认的数据库文件名（defaultDatabase）创建存储实例。
		// 这通常意味着存储将以文件形式存在于本地系统中。
		bot.storage, err = storage.FromFile(defaultDatabase)
		// 检查从文件创建存储实例时是否发生错误。
		if err != nil {
			// 如果有错误发生，返回 nil 和错误信息，中断初始化过程。
			return nil, err
		}
	}

	// 初始化订单控制器，管理订单的生命周期。
	bot.orderController = order.NewController(ctx, exch, bot.storage, bot.orderFeed)

	// 如果 Telegram 通知被启用，则设置并注册 Telegram 服务。
	if settings.Telegram.Enabled {
		//使用notification.NewTelegram函数来创建一个新的Telegram通知服务，这个函数接受两个参数：一个订单控制器（bot.orderController）和机器人的设置（settings）
		bot.telegram, err = notification.NewTelegram(bot.orderController, settings)
		if err != nil {
			return nil, err
		}
		// 注册 telegram 作为通知器。
		//意思就是说WithNotifier是一个函数传入设置好的bot.telegram实例作为参数，然后生成另一个函数然后返回type Option func(*NinjaBot)，又拿bot作为参数，然后执行
		WithNotifier(bot.telegram)(bot)
	}

	// 返回初始化完成的 NinjaBot 实例。
	return bot, nil
}

// 这段代码定义了一个 WithBacktest 函数，用于配置 NinjaBot 以便在回测模式下运行。回测模式是用于测试交易策略效果的一种模拟运行方式，通常用历史数据来模拟实时交易，以验证策略的可行性和效率。
// 这段代码确实是用来配置 NinjaBot 进行模拟回测的。通过设置 bot.backtest = true，它标识了机器人运行在回测模式，这意味着机器人将不执行真实的交易，而是使用历史数据来模拟交易，以评估交易策略的表现
func WithBacktest(wallet *exchange.PaperWallet) Option {
	return func(bot *NinjaBot) {
		// 设置机器人的 backtest 属性为 true，表示启动回测模式
		bot.backtest = true
		// 创建一个配置选项，使用指定的模拟钱包（PaperWallet）
		opt := WithPaperWallet(wallet)
		// 应用这个配置到机器人实例
		opt(bot)
	}
}

// WithStorage 设置机器人的存储接口，如果没有特别指定，它默认使用一个名为 ninjabot.db 的本地文件
func WithStorage(storage storage.Storage) Option {
	return func(bot *NinjaBot) {
		// 直接将传入的 storage 实例赋值给机器人的 storage 属性
		bot.storage = storage
	}
}

// WithLogLevel 设置日志级别。例如：log.DebugLevel, log.InfoLevel, log.WarnLevel, log.ErrorLevel, log.FatalLevel
func WithLogLevel(level log.Level) Option {
	return func(bot *NinjaBot) {
		// 调用 log 包的 SetLevel 方法设置全局日志级别
		log.SetLevel(level)
	}
}

// WithNotifier 为机器人注册一个通知器，目前支持电子邮件和Telegram通知
func WithNotifier(notifier service.Notifier) Option {
	// 返回一个符合 Option 类型（func(*NinjaBot)）的函数
	return func(bot *NinjaBot) {
		//  NinjaBot 实例能够使用这个 notifier 来发送通知，例如交易成功、交易失败等事件。
		bot.notifier = notifier
		// orderController 负责管理订单的生命周期（创建、修改、取消等）。通过设置它的 notifier，你确保了任何与订单相关的事件（如订单被创建或完成时）都会通知到 notifier，从而让最终用户能够接收到相关的更新。
		bot.orderController.SetNotifier(notifier)
		// 在 bot 实例上调用 SubscribeOrder 方法，并传入 notifier。这个方法使得 notifier 订阅订单的更新。这意味着任何订单状态的变更都会通过这个 notifier 发送通知。无论是订单的创建、执行、取消或任何其他更改，notifier 都会得到通知，并可据此向用户发送相关信息。
		bot.SubscribeOrder(notifier)
	}
}

// WithCandleSubscription 为给定的结构体订阅蜡烛图数据
func WithCandleSubscription(subscriber CandleSubscriber) Option {
	// 返回一个函数，这个函数符合 Option 类型定义（func(*NinjaBot)）。
	return func(bot *NinjaBot) {
		// 当返回的函数被调用时，它接收一个 *NinjaBot 实例作为参数，
		// 并且调用该实例的 SubscribeCandle 方法，将传入的 subscriber（实现了 CandleSubscriber 接口的对象）
		// 订阅到蜡烛图数据更新。这样，subscriber 将能够接收到K线数据的更新通知。
		bot.SubscribeCandle(subscriber)
	}
}

// WithPaperWallet 为机器人设置模拟钱包，用于回测和实时模拟。
func WithPaperWallet(wallet *exchange.PaperWallet) Option {
	// 返回一个函数，这个函数符合 Option 类型定义（func(*NinjaBot)）。
	return func(bot *NinjaBot) {
		// 将传入的模拟钱包实例赋值给机器人的 paperWallet 属性。
		// 这样，机器人就可以在进行交易模拟时使用这个模拟钱包来跟踪资金流动和交易结果。
		bot.paperWallet = wallet
	}
}

// SubscribeCandle 方法的主要作用是为 NinjaBot 实例注册一个或多个 CandleSubscriber 订阅者，以便它们可以接收特定交易对的K线（蜡烛图）数据更新。这使得订阅者能够基于这些数据更新来执行分析或交易策略
func (n *NinjaBot) SubscribeCandle(subscriptions ...CandleSubscriber) {
	// 遍历 NinjaBot 配置中定义的所有交易对。
	for _, pair := range n.settings.Pairs {
		// 遍历所有提供的订阅者。subscriptions 是一个 CandleSubscriber 类型的可变参数切片，
		// 允许一次性传入多个订阅者。
		for _, subscription := range subscriptions {
			// 调用 dataFeed 的 Subscribe 方法为每个交易对和每个订阅者注册通知。
			// 这个方法的参数是交易对、策略定义的时间帧、订阅者的 OnCandle 方法（当K线数据更新时会调用此方法），
			// 以及一个布尔值。布尔值在这里传入的是 false，可能表示某种处理模式（如是否立即处理或延迟处理等具体逻辑由实现决定）。
			n.dataFeed.Subscribe(pair, n.strategy.Timeframe(), subscription.OnCandle, false)
		}
	}
}

// WithOrderSubscription 为机器人添加一个订单更新的订阅者。
func WithOrderSubscription(subscriber OrderSubscriber) Option {
	// 返回一个符合 Option 类型（func(*NinjaBot)）的函数
	return func(bot *NinjaBot) {
		// 当返回的函数被调用时，它接收一个 *NinjaBot 实例作为参数，
		// 并且调用该实例的 SubscribeOrder 方法，将传入的 subscriber 注册为订单更新的订阅者。
		bot.SubscribeOrder(subscriber)
	}
}

// SubscribeOrder 为所有订单更新订阅提供的订阅者进行注册。
// 这段代码的主要作用是为 NinjaBot 实例中的订单数据流注册多个订阅者，使得这些订阅者能够接收到关于指定交易对订单状态更新的通知。具体来说，它允许机器人能够动态地为每一个交易对添加一个或多个订单更新监听者，这些监听者会在订单状态发生变化时接收到回调通知。这是处理多种交易策略和维持订单状态同步的关键功能，特别是在高频交易环境中尤为重要。
func (n *NinjaBot) SubscribeOrder(subscriptions ...OrderSubscriber) {
	// 遍历 NinjaBot 配置中定义的所有交易对。
	for _, pair := range n.settings.Pairs {
		// 遍历所有提供的订阅者。subscriptions 是一个 OrderSubscriber 类型的可变参数切片，
		// 允许一次性传入多个订阅者。
		for _, subscription := range subscriptions {
			// 它调用 orderFeed 的 Subscribe 方法为每个交易对和每个订阅者注册通知。
			// 这个方法的参数是交易对、订阅者的 OnOrder 方法（当订单更新时会调用此方法），以及一个布尔值。
			// 布尔值在这里传入的是 false，可能表示某种处理模式（如是否立即处理或延迟处理等具体逻辑由实现决定）。
			//订阅者的 OnOrder 方法，这是一个回调函数，每当相关的订单事件发生时（如订单创建、修改、取消等），这个方法就会被调用。
			n.orderFeed.Subscribe(pair, subscription.OnOrder, false)
		}
	}
}

// 通过调用 Controller() 这个方法，你可以获取到 NinjaBot 中的 orderController 组件。一旦获得了这个组件的访问权限，你就可以使用它提供的各种方法来管理订单，包括创建订单、取消订单、查看订单状态等。
func (n *NinjaBot) Controller() *order.Controller {
	return n.orderController
}

// Summary 方法显示所有交易、精度和一些机器人指标在标准输出上
// 要访问原始数据，可以使用 `bot.Controller().Results`
func (n *NinjaBot) Summary() {
	var (
		total  float64 // 这个变量用于累计机器人在所有交易中的总利润或总亏损。
		wins   int     // 这个变量记录了交易机器人执行的所有交易中，盈利交易的数量。
		loses  int     // 输的交易数
		volume float64 // 这个变量累计所有交易的总交易量
		//// SQN（系统质量数，评估交易系统性能的指标）,SQN = ✔交易次数 x 平均利差/标准差1.6 以下：系统表现较差，稳定性低 意思就是这个机器人系统性能差，交易输多赢少，1.6 至 2.0：表明交易系统具有基本的可行性，但性能表现尚未达到理想状态，2.0 至 2.5：表明交易系统表现良好，具有一定的稳定性和可靠性2.5 至 3.0：表明交易系统表现非常好，能够在多种市场环境中持续实现较高的盈利，3.0 以上：表明交易系统具有卓越的性能，能够在几乎所有市场条件下实现高盈利和低风险。这种系统通常具有很高的市场适应性和策略优势。
		sqn float64
	)
	//这行代码初始化了一个新的缓冲区（buffer），它用来暂存将要写入的数据，这里主要是用于存储表格数据。nil 表示初始化时没有任何数据。
	buffer := bytes.NewBuffer(nil)
	//使用 tablewriter.NewWriter 方法创建一个新的表格写入器，这个写入器将数据写入前面创建的 buffer 缓冲区。tablewriter 是一个流行的库，用于在Go程序中生成格式化的表格输出。
	table := tablewriter.NewWriter(buffer)
	//这行代码为表格设置列标题。每个字符串代表一列的标题，Pair: 交易对名称，Trades: 交易总数，Win: 获胜的交易数，Loss: 失败的交易数，% Win: 获胜比例，Payoff: 平均回报，Pr Fact.: 盈利因子，SQN: 系统质量数，Profit: 总利润，Volume: 交易量
	table.SetHeader([]string{"Pair", "Trades", "Win", "Loss", "% Win", "Payoff", "Pr Fact.", "SQN", "Profit", "Volume"})
	table.SetFooterAlignment(tablewriter.ALIGN_RIGHT) // 设置表尾对齐方式为右对齐
	avgPayoff := 0.0                                  // 加权回报贡献初始化0
	// 加权盈利因子贡献 盈利因子是交易系统盈利性能的一个关键指标。它通常通过比较总盈利和总亏损来计算。一个高于1的盈利因子表明系统总体上是盈利的，即盈利超过亏损。
	avgProfitFactor := 0.0

	returns := make([]float64, 0) // 创建一个用于存储回报率的切片
	// 遍历每个订单控制器中的结果汇总
	for _, summary := range n.orderController.Results {
		// 加权回报贡献=交易对的平均回报×(赢的次数+输的次数) ,加权回报贡献=平均回报率×总交易次数
		avgPayoff += summary.Payoff() * float64(len(summary.Win())+len(summary.Lose()))
		//加权盈利因子贡献=盈利因子×总交易次数
		avgProfitFactor += summary.ProfitFactor() * float64(len(summary.Win())+len(summary.Lose()))
		table.Append([]string{
			summary.Pair, // 交易对名称，如 "BTC/USD"
			strconv.Itoa(len(summary.Win()) + len(summary.Lose())), // 总交易次数，即赢的次数加上输的次数
			strconv.Itoa(len(summary.Win())),                       // 赢的交易次数
			strconv.Itoa(len(summary.Lose())),                      // 输的交易次数
			fmt.Sprintf("%.1f %%", float64(len(summary.Win()))/float64(len(summary.Win())+len(summary.Lose()))*100), // 胜率百分比，计算为赢的次数除以总交易次数，然后乘以100并保留一位小数
			fmt.Sprintf("%.3f", summary.Payoff()),       // 平均回报率，表示平均每笔交易的盈亏比率，保留三位小数
			fmt.Sprintf("%.3f", summary.ProfitFactor()), // 盈利因子，表示总盈利与总亏损的比率，也保留三位小数
			fmt.Sprintf("%.1f", summary.SQN()),          // 系统质量数（System Quality Number），衡量交易系统性能的指标，保留一位小数
			fmt.Sprintf("%.2f", summary.Profit()),       // 总利润，表示该交易对在统计期内的总体盈亏情况，保留两位小数
			fmt.Sprintf("%.2f", summary.Volume),         // 交易总量，通常指在该交易对上交易的资产总量，保留两位小数
		})

		// 更新总计数器
		total += summary.Profit()    // 将当前交易对的利润加到总利润上，累计整个交易系统的总利润。
		sqn += summary.SQN()         // 将当前交易对的系统质量数（SQN）加到总SQN上，累计整个交易系统的总SQN。
		wins += len(summary.Win())   // 将当前交易对的赢的次数加到总赢的次数上，累计整个交易系统的总胜场数。
		loses += len(summary.Lose()) // 将当前交易对的输的次数加到总输的次数上，累计整个交易系统的总负场数。
		volume += summary.Volume     // 将当前交易对的交易量加到总交易量上，累计整个交易系统的总交易量。

		// 将获胜的百分比回报率追加到返回列表中
		returns = append(returns, summary.WinPercent()...)
		// 将失败的百分比回报率追加到返回列表中
		returns = append(returns, summary.LosePercent()...)
	}

	// 设置并显示表格脚注，这段代码的作用是在表格的最底部添加一行，称为“脚注”，用以显示整个表格或数据集的汇总或总结性信息。就像注释一样
	// 设置并显示表格脚注
	table.SetFooter([]string{
		"TOTAL",                    // 脚注的第一列，标识这一行为总计行
		strconv.Itoa(wins + loses), // 总的交易次数（胜利次数加上失败次数）
		strconv.Itoa(wins),         // 总的胜利次数
		strconv.Itoa(loses),        // 总的失败次数
		fmt.Sprintf("%.1f %%", float64(wins)/float64(wins+loses)*100),    // 胜率百分比，计算方式是胜利次数除以总交易次数，然后乘以100并格式化为保留一位小数的百分比
		fmt.Sprintf("%.3f", avgPayoff/float64(wins+loses)),               // 加权平均回报率，计算方式是总的加权回报贡献除以总交易次数，并保留三位小数，高于1的值：表示整体上，投资或交易策略在考虑资本分配后是盈利的，即盈利超过了亏损。等于1的值：表示总的盈利和总的亏损相等，整体上处于盈亏平衡状态。低于1的值：表示亏损超过盈利，整体策略表现不佳。
		fmt.Sprintf("%.3f", avgProfitFactor/float64(wins+loses)),         // 加权平均盈利因子，计算方式是总的加权盈利因子贡献除以总交易次数，并保留三位小数，大于1：这表明总体上，策略或投资组合是盈利的，盈利超过了亏损。数值越高，表示盈利能力，等于1：表示盈利和亏损持平，总体上没有盈利也没有亏损，小于1：表示亏损超过盈利，策略或投资组合的表现不佳。
		fmt.Sprintf("%.1f", sqn/float64(len(n.orderController.Results))), // 平均系统质量数，计算方式是总SQN除以交易对的数量，并格式化为保留一位小数，SQN大于1.6：通常视为良好的交易系统。SQN在1.6到1.9之间：视为可接受的交易系统。N大于2.0：视为优秀的交易系统，特别是当SQN超过2.5甚至3.0时，表明交易系统非常优秀，具有很高的盈利能力和稳定性。
		fmt.Sprintf("%.2f", total),                                       // 总利润，显示为保留两位小数的格式
		fmt.Sprintf("%.2f", volume),                                      // 总交易量，显示为保留两位小数的格式
	})
	table.Render() // table.Render() 函数的作用是将之前设置的表格配置，包括添加的所有行和列、格式化选项、脚注等，进行最终的渲染处理，使其在指定的输出（通常是控制台或文件）中以表格形式展示出来。

	//这一行打印之前使用 tablewriter 创建的表格内容。表格数据被存储在一个名为 buffer 的缓冲区中，buffer.String() 将这个缓冲区的内容转换为字符串格式，随后通过 fmt.Println 打印到控制台。
	fmt.Println(buffer.String())
	//分隔符
	fmt.Println("------ RETURN -------")
	//这个变量用于累加整个交易策略或交易系统中所有交易的回报率。
	totalReturn := 0.0
	//这行代码初始化一个float64类型的切片returnsPercent，其长度与returns切片的长度相同。这个切片用来存储转换为百分比形式的回报率。
	returnsPercent := make([]float64, len(returns))
	//这段代码的主要功能是遍历returns切片中的每个元素（每个元素代表一个特定的回报率）将每个回报率p（通常表示为小数形式，比如0.05表示5%的回报）乘以100，将其转换为百分比形式。例如，如果p是0.05，则p*100等于5.0，表示5%的回报
	for _, p := range returns {
		returnsPercent = append(returnsPercent, p*100)
		//累加每个交易的回报率到totalReturn变量中。这个变量用于计算所有交易回报率的总和。
		totalReturn += p
	}
	//就是[]float64{5.0, 7.5, 5.0, 10.0, -2.5, 0.0, 3.0, 8.0, 12.0, -1.0, 4.0, 6.5, 11.0, 2.0, -3.0}  如果我们将这个范围(-3.0到12.0)分成15个等宽的柱体，每个柱体的宽度为(12.0 - (-3.0))/15 ≈ 1.0。这意味着每个箱覆盖了大约1个百分点的范围，分布从-3.0开始，每个箱宽约为1.0，一直到12.0，直方图每个柱体的高度表示落在对应值范围内的数据点数量。 如果某个区间的柱体特别高，这表明许多数据点落在这个区间内。反之，如果柱体较低，表示该区间的数据点较少
	//histogram.Hist 是一个函数，通常是在某个统计或数据可视化库中定义的。这个函数的目的是根据提供的数据创建一个直方图的数据结构。第一个参数 15 指的是要将数据分成多少个箱（bins）。在这个案例中，数据集将被分成15个等宽的区间。第二个参数 returnsPercent 是一个包含数据点的切片（slice），这些数据点是我们要为其创建直方图的实际数值。
	hist := histogram.Hist(15, returnsPercent) // 使用直方图表示返回数据
	//istogram.Fprint 是一个函数，其作用是将直方图数据格式化并打印到指定的输出流。os.Stdout：这是一个输出流，指向标准输出（通常是命令行或控制台）。使用这个参数意味着直方图将被打印到命令行。hist：这是直方图的数据结构，包含了直方图的所有信息，如每个箱的数据点计数等。histogram.Linear(10)：这是一个格式化选项，通常这个函数调用会返回一个配置对象，设置直方图的某些显示特性。Linear(10) 可能意味着直方图的某些线性配置，如标签间距或比例尺，具体含义取决于histogram包的实现，数字10可能表示某种分辨率或宽度设置。
	histogram.Fprint(os.Stdout, hist, histogram.Linear(10)) // 在标准输出上打印直方图
	//它的作用是在直方图打印完成后添加一个空行，使得输出更加整洁，与随后的输出内容分开。
	fmt.Println()

	fmt.Println("------ CONFIDENCE INTERVAL (95%) -------")
	//遍历每个交易对的交易结果，拿到交易对，还有交易结果
	for pair, summary := range n.orderController.Results {
		//打印交易对
		fmt.Printf("| %s |\n", pair)
		//所有赢得百分比，与多个失败百分比结合起来，...这个是展开符，因为appent不支持两个切片进行合并，只能是一个切片召开之后，里面的数据合并到另一个切片当中
		returns := append(summary.WinPercent(), summary.LosePercent()...)
		// 使用 bootstrap 方法计算平均回报率的 95% 置信区间，将赢的和输的回报率切片returns，重采样1000次(就是从切片里面重新抽取一样的数量) 然后每一次重采样都拿到一个平均值，然后拿到10000个平均值，从低到高排序，低于5%的不算，看5%-95%在什么区间，真实的平均回报率就在哪个区间
		returnsInterval := metrics.Bootstrap(returns, metrics.Mean, 10000, 0.95)
		// 从 returns 数据集中分别提取正收益和负收益，（包括正收益和负收益一起）进行重采样10000次，分别计算它们的平均比率。把每次计算的平均比率从低到高排序，取5%-95%区间，真实的Payoff值（即盈亏比率的绝对值）将落在这个区间内
		payoffInterval := metrics.Bootstrap(returns, metrics.Payoff, 10000, 0.95)
		// 从returns 数据集拿到所有交易的回报率，其中正数代表盈利，负数代表亏损，然后进行重采样10000次，每次采样都计算利润因子，重低到高进行排序，取5%-95%的值。真实的利润因子（盈利和亏损的比率）将落在这个区间内
		profitFactorInterval := metrics.Bootstrap(returns, metrics.ProfitFactor, 10000, 0.95)
		// 打印表示平均回报率的均值乘以100，转换成百分比形式。 分别表示置信区间的下限和上限，也转换成百分比形式。
		fmt.Printf("RETURN:      %.2f%% (%.2f%% ~ %.2f%%)\n",
			returnsInterval.Mean*100, returnsInterval.Lower*100, returnsInterval.Upper*100)
		//打印 Payoff值的均值，Payoff值的置信区间的下限和上限。
		fmt.Printf("PAYOFF:      %.2f (%.2f ~ %.2f)\n",
			payoffInterval.Mean, payoffInterval.Lower, payoffInterval.Upper)
		// 打印利润因子的均值，利润因子的置信区间的下限和上限。
		fmt.Printf("PROF.FACTOR: %.2f (%.2f ~ %.2f)\n",
			profitFactorInterval.Mean, profitFactorInterval.Lower, profitFactorInterval.Upper)
	}
	fmt.Println()

	// 如果配置了模拟钱包，则调用其总结方法以显示模拟钱包的总结信息
	if n.paperWallet != nil {
		n.paperWallet.Summary()
	}
}

// SaveReturns 保存每个交易对的交易结果到CSV文件。
// outputDir 指定了保存文件的目录。
// 如果 outputDir 是 /path/to/folder 且 summary.Pair 是 "BTC/USD"，则 outputFile 将被设置为 /path/to/folder/BTC/USD.csv。这个路径之后用于保存该交易对的交易结果数据到一个CSV文件中。
func (n NinjaBot) SaveReturns(outputDir string) error {
	// 遍历orderController中存储的所有交易结果
	for _, summary := range n.orderController.Results {
		// 为每个交易对生成CSV文件的路径
		outputFile := fmt.Sprintf("%s/%s.csv", outputDir, summary.Pair)
		// 调用summary对象的SaveReturns方法将交易数据保存到CSV文件，如果有错误发生则返回错误
		if err := summary.SaveReturns(outputFile); err != nil {
			return err
		}
	}
	// 如果所有文件都成功保存，则返回nil表示无错误
	return nil
}

// onCandle 处理接收到的单个K线数据，并将其放入优先队列。
// candle 是接收到的K线数据。
// 意思就是将新接收到的k线数据进行排列，这种优先级可能基于时间戳，即确保最早的数据先被处理。也可能基于其他因素，比如交易量大小、价格变动的幅度等。放到新的位置
func (n *NinjaBot) onCandle(candle model.Candle) {
	// 将K线数据推入优先队列，优先队列可能基于时间或其他标准排序
	n.priorityQueueCandle.Push(candle)
}

// processCandle 处理从队列中获取的K线数据，更新钱包和策略控制器的状态。
// candle 是从队列中获取的K线数据。
func (n *NinjaBot) processCandle(candle model.Candle) {
	// 如果虚拟钱包paperWallet实例存在，传入真实的k线数据更新它的状态，如检查订单状态，更新订单状态和交易量，更新虚拟账户中的资产数量和平均价格 等
	if n.paperWallet != nil {
		n.paperWallet.OnCandle(candle)
	}

	// 更新对应交易对的策略控制器，处理部分完成的K线，K线数据通常在特定时间间隔结束时被认为是完成的，比如一分钟、一小时等。但在实际交易中，可能需要在K线完全形成之前做出反应，特别是在高频交易或某些需要快速响应市场变动的策略中。OnPartialCandle 方法就是用于这种情况，它允许策略在K线数据还在形成中时就开始处理和分析这些数据。通过这种方法，交易机器人可以更快地响应市场变化，不必等到完整的K线数据形成后才做出决策。这对于捕捉短暂的市场机会尤其重要。
	n.strategiesControllers[candle.Pair].OnPartialCandle(candle)
	// 如果K线完全完成（比如时间完结或达到其它完成条件）
	if candle.Complete {
		// 更新策略控制器状态，处理完整的K线，这个方法是根据k线完全形成而分析的，策略控制器可能会分析K线数据的特征（如价格变动、交易量等），并根据策略算法决定是否保持当前持仓、买入或卖出。例如，策略可能会在检测到价格突破支持线时决定买入。
		n.strategiesControllers[candle.Pair].OnCandle(candle)
		// 让订单控制器处理完整的K线，更新收盘价
		n.orderController.OnCandle(candle)
	}
}

// 处理缓存中待处理的K线数据
func (n *NinjaBot) processCandles() {
	// 遍历从优先队列中弹出的每一个项
	for item := range n.priorityQueueCandle.PopLock() {
		//当需要从队列中弹出元素时，这个回调函数会被触发。回调函数负责调用 Pop() 方法来从队列中实际移除优先级最高的元素，并将其发送到一个通道 (ch)。rocessCandles() 方法监听从 PopLock() 方法返回的通道。当通道中出现数据（即K线数据）时，processCandles() 方法会接收这些数据，并将每个数据项（此处为 K线数据）处理转换为 model.Candle 类型，然后进一步处理这些数据，更新策略控制器和虚拟钱包的状态
		n.processCandle(item.(model.Candle))
	}
}

// 开始回测过程并创建进度条
func (n *NinjaBot) backtestCandles() {
	// 记录开始回测的日志信息
	log.Info("[SETUP] Starting backtesting")

	// 创建一个进度条，长度设置为优先队列中的元素数量
	progressBar := progressbar.Default(int64(n.priorityQueueCandle.Len()))

	// 当优先队列中还有元素时，持续处理
	for n.priorityQueueCandle.Len() > 0 {
		// 从优先队列中弹出一个元素
		item := n.priorityQueueCandle.Pop()

		// 将弹出的元素类型断言为model.Candle
		candle := item.(model.Candle)

		// 如果虚拟钱包存在，则使用当前的K线数据更新虚拟钱包的状态
		if n.paperWallet != nil {
			n.paperWallet.OnCandle(candle)
		}

		// 更新对应交易对的策略控制器，处理部分完成的K线
		n.strategiesControllers[candle.Pair].OnPartialCandle(candle)

		// 如果K线数据已经完整，处理完整的K线
		if candle.Complete {
			n.strategiesControllers[candle.Pair].OnCandle(candle)
		}

		// 更新进度条，每处理完一个K线数据，进度条增加1
		if err := progressBar.Add(1); err != nil {
			// 如果更新进度条失败，记录警告日志
			log.Warnf("update progressbar fail: %v", err)
		}
	}
}

// 在NinjaBot启动之前，我们需要加载必要的数据来填充策略指标
// 然后，我们需要获取时间框架和预热期以获取必要的K线数据
func (n *NinjaBot) preload(ctx context.Context, pair string) error {
	// 如果是在回测模式下，不需要预加载数据，直接返回
	if n.backtest {
		return nil
	}

	// 从交易所获取限定数量的K线数据，数量由策略的时间框架和预热期决定
	candles, err := n.exchange.CandlesByLimit(ctx, pair, n.strategy.Timeframe(), n.strategy.WarmupPeriod())
	if err != nil {
		// 如果获取数据时出现错误，返回错误
		return err
	}

	// 遍历获取到的K线数据，处理每根K线
	for _, candle := range candles {
		n.processCandle(candle)
	}

	// 将获取到的K线数据预加载到数据提供者中，为策略指标填充数据
	n.dataFeed.Preload(pair, n.strategy.Timeframe(), candles)

	// 返回无错误，表示预加载成功
	return nil
}

// Run 会初始化策略控制器、订单控制器、预加载数据并启动机器人
func (n *NinjaBot) Run(ctx context.Context) error {
	// 遍历设定中的所有交易对
	for _, pair := range n.settings.Pairs {
		// 设置并订阅策略到数据源（K线数据）
		n.strategiesControllers[pair] = strategy.NewStrategyController(pair, n.strategy, n.orderController)

		// 为预热期预加载K线数据
		err := n.preload(ctx, pair)
		if err != nil {
			return err // 如果预加载过程中出现错误，返回错误并终止
		}

		// 作用是订阅指定的交易对和时间框架到一个数据源，使得每当有新的K线数据到S来时，就会立即调用 n.onCandle 函数来处理这些数据，而不需等待K线完全关闭。这允许策略能够快速响应市场变化，从而及时执行交易决策。
		n.dataFeed.Subscribe(pair, n.strategy.Timeframe(), n.onCandle, false)

		// 启动策略控制器就是为每个交易对激活对应的交易策略，使其能够开始监测市场并执行交易操作
		n.strategiesControllers[pair].Start()
	}

	// 启动订单数据流，订单数据流可能来自于交易所或者其他交易平台，它会持续地提供新的订单信息给交易机器人。
	n.orderFeed.Start()
	//启动了订单控制器。订单控制器是负责接收、处理和执行订单的组件。一旦启动，订单控制器就开始监听来自订单数据流的订单信息，并根据预先设定的策略执行相应的交易操作。
	n.orderController.Start()
	defer n.orderController.Stop() // 确保在函数退出时停止订单控制器

	// 如果配置了Telegram通知，启动 Telegram 服务，以便在交易机器人执行交易或发生其他重要事件时发送通知。
	if n.telegram != nil {
		n.telegram.Start()
	}

	// 启动数据流，就是可以源源不断的从交易所中拿到新k数据，给机器人处理
	n.dataFeed.Start(n.backtest)

	// 如果当前处于回测环境，则调用 n.backtestCandles() 方法来执行回测过程。在回测过程中，机器人会按照历史数据的时间顺序逐步回放，并根据策略逻辑进行交易决策，从而模拟真实市场环境下的交易情况。
	if n.backtest {
		n.backtestCandles()
	} else {
		//n.backtest 为 false，表示当前处于生产环境，则调用 n.processCandles() 方法来处理实时K线数据。在生产环境中，机器人会不断地接收实时市场数据，并根据最新的K线数据进行实时的交易决策和操作。
		n.processCandles()
	}

	// 无错误返回，表示启动流程正常完成
	return nil
}
