package notification

// 引入所需的包
import (
	"errors"  // 用于处理错误
	"fmt"     // 用于格式化输出
	"regexp"  // 正则表达式支持
	"strconv" // 字符串和基本类型之间转换
	"strings" // 字符串操作
	"time"    // 时间操作

	log "github.com/sirupsen/logrus" // 强大的日志记录库
	tb "gopkg.in/tucnak/telebot.v2"  // Telegram机器人库

	// ninjabot特定的包，用于交易和交易信息
	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/order"
	"github.com/rodrigo-brito/ninjabot/service"
)

/*
这段代码实现了一个Telegram机器人，用于通过Telegram应用程序与一个交易系统（如ninjabot）进行交互。它允许用户通过发送特定的Telegram命令来执行交易操作（如买卖货币）、查询账户余额、查看交易状态和利润等。机器人还能在某些交易事件发生时（例如订单完成、出现错误等）主动向用户发送通知。
*/

// 定义了两个正则表达式用于匹配用户的买卖命令
var (
	// 匹配购买命令的正则表达式，捕获交易对、金额和百分比（可选）
	/*
		buyRegexp 解释
		/buy\s+：匹配以/buy开头的字符串，后面跟着至少一个空白字符（\s+）。
		(?P<pair>\w+)：匹配交易对，并将其命名为pair。\w+匹配一个或多个字母数字字符，代表交易对（如BTCUSDT）。
		(?P<amount>\d+(?:\.\d+)?)：匹配金额，并将其命名为amount。\d+匹配一个或多个数字，(?:\.\d+)?是一个非捕获组，匹配可能存在的小数点和小数部分，整个表达式代表匹配整数或小数金额。
		(?P<percent>%)?：可选地匹配一个百分号，并将其命名为percent。表示用户可能指定金额为百分比。
	*/
	buyRegexp = regexp.MustCompile(`/buy\s+(?P<pair>\w+)\s+(?P<amount>\d+(?:\.\d+)?)(?P<percent>%)?`)
	// 匹配出售命令的正则表达式，捕获交易对、金额和百分比（可选）
	/*
		sellRegexp 解释
		/sell\s+：与/buy\s+类似，匹配以/sell开头的字符串，后面至少有一个空白字符。
		(?P<pair>\w+)、(?P<amount>\d+(?:\.\d+)?)、(?P<percent>%)?：与buyRegexp中的对应部分相同，分别用于匹配交易对、金额和可选的百分比标记。
	*/
	sellRegexp = regexp.MustCompile(`/sell\s+(?P<pair>\w+)\s+(?P<amount>\d+(?:\.\d+)?)(?P<percent>%)?`)
)

// telegram结构体定义了Telegram机器人所需的核心属性
type telegram struct {
	// 存储机器人设置（如授权的用户ID等），存储了机器人的设置信息，例如授权的用户 ID 等。这些设置决定了机器人的行为，比如哪些用户可以使用机器人，机器人的默认行为等。
	settings model.Settings
	//控制订单的创建、查询和管理。这个字段负责处理与订单相关的逻辑，比如用户下单后的处理过程，订单的状态管理等。
	orderController *order.Controller
	//这个字段代表默认的 Telegram 键盘菜单。键盘菜单是用户与机器人交互时显示的界面元素，通常包含各种命令按钮或选项，用户可以通过点击按钮来执行相应的操作。在这里，defaultMenu 存储了默认的键盘菜单，以便在需要时向用户展示。
	defaultMenu *tb.ReplyMarkup
	//这个字段是 Telegram 机器人客户端实例，用于与 Telegram 服务器进行通信的接口。这个实例负责接收用户的消息、发送消息给用户，以及处理与 Telegram 服务器的其他交互。在实际使用中，可以通过这个客户端实例来监听用户的消息，回复用户的消息，以及执行其他与 Telegram 服务器相关的操作，如设置命令、更新菜单等。
	client *tb.Bot
}

type Option func(telegram *telegram)

// NewTelegram 函数用于创建一个新的 Telegram 服务实例。
// 参数 controller 是订单控制器实例，用于控制订单的创建、查询和管理。参数 settings 是机器人的设置信息，包括授权用户ID等，参数 options 是可选的一系列选项，用于定制 Telegram 服务，返回一个 service.Telegram 接口实例和一个可能的错误。
// 这段代码的作用是创建并初始化一个基于 Telegram 的自动化交易机器人，它能实时响应授权用户的交易指令和查询请求
func NewTelegram(controller *order.Controller, settings model.Settings, options ...Option) (service.Telegram, error) {
	// 创建默认菜单，默认菜单是一个 Telegram 机器人用于与用户交互的界面元素，其中包含了按钮、键盘等，可以通过点击按钮或输入指令与机器人进行交互。默认的意思就是里面的设置保持最初设置的样子
	menu := &tb.ReplyMarkup{ResizeReplyKeyboard: true}
	// 创建长轮询器，长轮询（Long Polling）的作用是实现实时消息更新的机制。通过设置超时时间，就是机器人可能会发送下单、取消订单等操作的请求，以进行交易操作后，长轮询器可以在一定时间内等待服务器的响应。如果在超时时间内服务器有新消息，则立即返回该消息；如果超时时间内没有新消息，则返回空响应
	//设置超时时间10秒钟，10秒钟没有收到服务器响应，就返回空响应
	poller := &tb.LongPoller{Timeout: 10 * time.Second}

	// 这段代码创建了一个用户中间件，用于检查用户是否已经授权。它使用了Telegram库中的NewMiddlewarePoller函数，该函数需要传入一个轮询器（poller）和一个回调函数作为参数。
	userMiddleware := tb.NewMiddlewarePoller(poller, func(u *tb.Update) bool {
		//是检查发送者的信息是否为空，发送者是否为空，如果为空，就执行下面的操作
		//在Telegram Bot API中，消息通常是通过更新（update）的形式传递的，更新可以包含各种事件，例如消息、新成员加入、成员离开等。
		if u.Message == nil || u.Message.Sender == nil {
			log.Error("no message, ", u)
			return false
		}

		// 检查发送者是否为授权用户
		//遍历出接收通知的用户id列表
		for _, user := range settings.Telegram.Users {
			//检查消息中的发送者ID是否存在于这个列表中。如果存在，则返回 true，表示发送者是授权用户，否则返回 false
			if int(u.Message.Sender.ID) == user {
				return true
			}
		}
		//如果没有在列表中，则显示无效用户，返回false
		log.Error("invalid user, ", u.Message)
		return false
	})

	//这部分代码是创建了一个Telegram客户端实例，用于与Telegram服务器进行通信。
	client, err := tb.NewBot(tb.Settings{
		// 表明机器人发送的消息将解析 Markdown 格式，使得消息可以包含格式化文本（如粗体、斜体等）。
		ParseMode: tb.ModeMarkdown,
		//指定了机器人的访问令牌（API Token），这是一个必需的认证凭证，用于验证机器人身份并授权它与 Telegram 服务器通信。这个令牌在创建机器人时由 Telegram 提供，并需要保密。
		//这个 token 是在 Telegram 应用内通过交云 BotFather 机器人来获取的。BotFather 是 Telegram 官方提供的一个机器人，用于创建和管理机器人
		Token: settings.Telegram.Token,
		// 设置了机器人如何接收消息的机制。这里采用的是用户中间件作为轮询器，意味着所有通过长轮询接收到的消息都会先经过这个中间件进行处理。用户中间件可以对消息进行筛选，例如检查消息发送者是否是授权用户，只有通过检查的消息才会被机器人进一步处理。
		Poller: userMiddleware,
	})
	if err != nil {
		return nil, err
	}

	// 创建默认菜单按钮，这段代码定义了一组按钮，这些按钮用于构建 Telegram 机器人的交云菜单。每个按钮都与一个特定的命令相关联，用户可以通过点击这些按钮来快速发送对应的命令给机器人
	//创建交云菜单按钮的目的是在机器人的聊天界面中提供可视化的按钮，用户可以直接点击这些按钮来发送对应的命令。这种方式不需要用户记忆或手动输入命令文本，提高了用户的交云便利性和体验。
	var (
		statusBtn  = menu.Text("/status")  // 创建 "/status" 按钮，用于显示机器人当前状态
		profitBtn  = menu.Text("/profit")  // 创建 "/profit" 按钮，用于查询最近的交易盈利
		balanceBtn = menu.Text("/balance") // 创建 "/balance" 按钮，用于查询账户余额
		startBtn   = menu.Text("/start")   // 创建 "/start" 按钮，用于开始机器人服务或交易
		stopBtn    = menu.Text("/stop")    // 创建 "/stop" 按钮，用于停止机器人服务或交易
		buyBtn     = menu.Text("/buy")     // 创建 "/buy" 按钮，用于执行购买或买入交易
		sellBtn    = menu.Text("/sell")    // 创建 "/sell" 按钮，用于执行出售或卖出交易
	)

	// 这这段代码是为 Telegram 机器人客户端设置命令。每个命令由两部分组成：命令文本和描述。命令文本是用户需要输入的文本（如 /help），而描述则是对命令作用的简短说明
	//就是第一个设置描述的时候就相当让人们知道这个命令是拿来做什么的，相当于一个标注，第二个设置，是真的通过命令然后执行操作
	err = client.SetCommands([]tb.Command{
		{Text: "/help", Description: "显示帮助指令"},
		{Text: "/stop", Description: "停止买卖交易"},
		{Text: "/start", Description: "开始买卖交易"},
		{Text: "/status", Description: "检查机器人状态"},
		{Text: "/balance", Description: "钱包余额"},
		{Text: "/profit", Description: "最近交易结果摘要"},
		{Text: "/buy", Description: "开设买单"},
		{Text: "/sell", Description: "开设卖单"},
	})
	if err != nil {
		return nil, err
	}

	// 设置默认菜单按钮
	//Telegram 机器人的默认交云菜单，具体来说，它设置了一个自定义的回复键盘，其中包含了之前创建的按钮通过这种方式，用户界面变得更加友好，用户不需要记住具体的命令文本，而是可以通过点击按钮来交云。
	//enu.Reply 函数用于设置 Telegram 机器人的自定义回复键盘。这个键盘由一系列按钮组成，每个按钮都对应一个命令。当用户点击这些按钮时，就会向机器人发送与按钮相关联的命令。
	menu.Reply(
		//这一行定义了键盘的第一行，包含三个按钮：“/status”，“/balance”，和“/profit”。用户点击这些按钮，就会向机器人发送对应的命令。
		menu.Row(statusBtn, balanceBtn, profitBtn),
		//这一行定义了键盘的第二行，同样方式包含了“/start”，“/stop”，“/buy”，和“/sell”按钮。
		menu.Row(startBtn, stopBtn, buyBtn, sellBtn),
	)

	// 这段代码首先创建了一个 telegram 结构体实例，并用之前配置好的信息（如控制器、客户端、设置和菜单）来初始化这个实例。这样，telegram 实例就包含了所有必要的组件和配置，使其能够按预期工作
	bot := &telegram{
		orderController: controller, //责处理所有订单相关逻辑的控制器
		client:          client,     //这是设置好的 Telegram 客户端，用于与 Telegram API 通信。
		settings:        settings,   //这包含了机器人运行所需的各种设置信息
		defaultMenu:     menu,       //这是机器人的默认交云菜单，包含用户可以点击的按钮，以方便用户操作。
	}

	// 应用所有选项配置
	//这个调用过程使得每个 Option 函数能够对 bot（你的 Telegram 机器人实例）进行配置或修改。
	for _, option := range options {
		option(bot)
	}

	// 设置命令处理函数
	//就是第一个设置描述的时候就相当让人们知道这个命令是拿来做什么的，相当于一个标注，第二个设置，是真的通过命令然后执行操作
	client.Handle("/help", bot.HelpHandle)
	client.Handle("/start", bot.StartHandle)
	client.Handle("/stop", bot.StopHandle)
	client.Handle("/status", bot.StatusHandle)
	client.Handle("/balance", bot.BalanceHandle)
	client.Handle("/profit", bot.ProfitHandle)
	client.Handle("/buy", bot.BuyHandle)
	client.Handle("/sell", bot.SellHandle)

	return bot, nil
}

func (t telegram) Start() {
	// 启动 Telegram 客户端的 goroutine。
	//启动一个协程，机器人可以单独运行，不会受其他影响
	go t.client.Start()
	// 向配置中指定的每个 Telegram 用户发送消息，通知他们机器人已初始化，并附上默认菜单。
	for _, id := range t.settings.Telegram.Users {
		// 通过指定的用户 ID 创建一个 User 结构，并向该用户发送消息 "Bot initialized."，同时附带默认菜单。
		_, err := t.client.Send(&tb.User{ID: int64(id)}, "Bot initialized.", t.defaultMenu)
		// 如果发送消息时出现错误，记录错误日志。
		if err != nil {
			log.Error(err)
		}
	}
}

// Notify 方法是在 telegram 类型的结构体中定义的，用于向所有配置的用户发送文本消息。
func (t telegram) Notify(text string) {
	// 这行代码开始一个循环，遍历 t.settings.Telegram.Users 列表。这个列表包含了所有配置好的用户ID，表示需要接收消息的用户。
	for _, user := range t.settings.Telegram.Users {
		// 创建一个 tb.User 结构体，这是 Telegram bot 库使用的用户标识结构，其中 ID 是用户的唯一标识符
		//利用sand方法发送通知给在列表里面的用户
		_, err := t.client.Send(&tb.User{ID: int64(user)}, text)
		// 使用 t.client.Send 方法发送 text 给指定的用户
		// 如果发送过程中出现错误
		if err != nil {
			// 记录错误
			log.Error(err)
		}
	}
}

// 这个函数用来处理 "/balance" 命令。它首先获取账户信息，然后计算每个交易对的资产价值，并将资产和报价的信息以及总价值发送给消息的发送者。
func (t telegram) BalanceHandle(m *tb.Message) {
	// 函数开始时定义了一个字符串message，以“BALANCE”为标题，用于存储并最终显示账户的余额信息。
	//星号（*）用于标记文本为粗体。所以 "*BALANCE*" 这部分文本在发送到用户的 Telegram 消息中会以粗体显示，强调“BALANCE”这个词，即“余额”。后面的 \n 是一个换行符，表示在这个标题后面的内容应该开始于新的一行。这样做可以提高消息的可读性，让用户一眼就能看到这是一个关于账户余额的信息总结。
	message := "*BALANCE*\n"
	// 使用一个map（名为quotesValue）来存储每种报价货币的价值，以及一个浮点数total来累计所有资产的总价值。
	quotesValue := make(map[string]float64)
	total := 0.0

	// 获取账户信息
	account, err := t.orderController.Account()
	if err != nil {
		log.Error(err)
		t.OnError(err)
		return
	}

	// 遍历所有交易对
	for _, pair := range t.settings.Pairs {
		// 将交易对分为基础资产，报价资产
		assetPair, quotePair := exchange.SplitAssetQuote(pair)
		// 获取基础资产和报价资产的余额信息
		assetBalance, quoteBalance := account.Balance(assetPair, quotePair)

		// 计算基础资产和报价资产的总量
		assetSize := assetBalance.Free + assetBalance.Lock
		quoteSize := quoteBalance.Free + quoteBalance.Lock

		// 获取最新的报价
		quote, err := t.orderController.LastQuote(pair)
		if err != nil {
			log.Error(err)
			t.OnError(err)
			return
		}

		// 计算基础资产的价值 = 基础资产数量 x 报价资产
		assetValue := assetSize * quote
		// 将报价的价值存储到 map 中
		quotesValue[quotePair] = quoteSize
		// 累加总价值= 原有账户 + 基础资产的价值
		total += assetValue
		// 将基础资产交易对，资产数量，基础资产价值和报价的信息添加到消息字符串中，这表示将新的内容追加到已存在的message字符串变量中。
		//就是添加到文本的末尾message := "I have a " message += "dream." fmt.Println(message) // 输出：I have a dream.
		message += fmt.Sprintf("%s: `%.4f` ≅ `%.2f` %s \n", assetPair, assetSize, assetValue, quotePair)
	}

	// 遍历报价的价值 如usdt，并将报价的价值添加到消息字符串中
	for quote, value := range quotesValue {
		//把报价资产放到总总产里面
		total += value

		message += fmt.Sprintf("%s: `%.4f`\n", quote, value)
	}

	// 添加总价值信息到消息字符串中
	message += fmt.Sprintf("-----\nTotal: `%.4f`\n", total)

	// 将消息发送给消息发送者
	_, err = t.client.Send(m.Sender, message)
	if err != nil {
		log.Error(err)
	}
}

// HelpHandle 函数是一个 Telegram 机器人的命令处理器，专门用来处理 /help 命令。当用户发送 /help 命令时，这个函数会被调用来显示所有可用的机器人命令及其描述。
// t.client.GetCommands() 这个函数的作用是从 Telegram 机器人的客户端获取所有已注册的命令和它们的描述。这些命令通常是在机器人启动时，或在特定配置函数中使用类似于 client.SetCommands([]tb.Command{...}) 这样的调用来注册的。
func (t telegram) HelpHandle(m *tb.Message) {
	// 尝试从 Telegram 客户端获取已注册的所有命令和它们的描述
	commands, err := t.client.GetCommands()
	if err != nil {
		// 如果获取命令时出错，则记录错误并调用错误处理函数
		log.Error(err)
		t.OnError(err)
		return // 由于发生错误，提前终止函数执行
	}

	// 创建一个字符串切片，用于存储每个命令的文本表示，切片的初始容量设为命令的数量
	lines := make([]string, 0, len(commands))
	for _, command := range commands {
		// 遍历所有命令，将每个命令还有命令描述信息格式化化为 "/command - description" 的形式，并追加到切片中
		// 例如：/help - 显示帮助指令
		lines = append(lines, fmt.Sprintf("/%s - %s", command.Text, command.Description))
	}

	// 将所有命令的描述连接成一个字符串，每个命令占一行，然后发送这个字符串给请求帮助的用户
	/*
			strings.Join 函数确实是将 lines 切片中的所有命令合并成一个单一的字符串，并用换行符 ("\n")
		例如:/help - 显示帮助信息
			/start - 启动机器人
			/stop - 停止机器人
			/status - 显示当前状态

	*/
	//当用户发送帮助信息时，用户的信息会贮存在m.Sender里面，m.Sender 包含了发送消息给机器人的用户的信息，通常包括用户的 ID 和其他可能的用户信息。当用户发送一个消息（比如请求帮助信息）到机器人时，机器人通过访问 m.Sender 能获取到这个用户的具体信息。然后通过t.client.Send返回请求的信息给发送者
	_, err = t.client.Send(m.Sender, strings.Join(lines, "\n"))
	if err != nil {
		// 如果发送消息时发生错误，则记录错误
		log.Error(err)
	}
}

// ProfitHandle函数是处理特定于Telegram机器人的“查看利润”命令的功能
func (t telegram) ProfitHandle(m *tb.Message) {
	// 检查是否有交易结果记录。如果没有，向用户发送一条消息，并返回。
	if len(t.orderController.Results) == 0 {
		_, err := t.client.Send(m.Sender, "No trades registered.")
		// 如果发送消息时出现错误，记录错误信息。
		if err != nil {
			log.Error(err)
		}
		// 由于没有交易数据，函数提前返回。
		return
	}

	// 遍历所有交易对的结果。
	//市场订单很好的执行这个/buy命令 因为市场订单快速执行， 我发送命令之后直接就买
	for pair, summary := range t.orderController.Results {
		// 对每个交易对，发送一条包含交易对名称和交易摘要的消息。
		_, err := t.client.Send(m.Sender, fmt.Sprintf("*PAIR*: `%s`\n`%s`", pair, summary.String()))
		// 如果发送消息时出现错误，记录错误信息。
		if err != nil {
			log.Error(err)
		}
	}
}

// BuyHandle 处理通过 Telegram 消息接收到的购买命令。
func (t telegram) BuyHandle(m *tb.Message) {
	//我们定义了一个正则表达式，然后用户发送文本：用户通过 Telegram 发送一条消息，比如 /buy BTCUSDT 100 我们通过FindStringSubmatch 方法捕捉信息，如果符合了我们定义的正则表达式，就会放在m.text， ，这些消息以切片形式返回到match
	match := buyRegexp.FindStringSubmatch(m.Text)
	/*
		// 如果没有匹配结果，发送错误消息给用户并返回
		Invalid command.
		Examples of usage:
		`/buy BTCUSDT 100`
		`/buy BTCUSDT 50%`

	*/
	if len(match) == 0 {
		_, err := t.client.Send(m.Sender, "Invalid command.\nExamples of usage:\n`/buy BTCUSDT 100`\n\n`/buy BTCUSDT 50%`")
		if err != nil {
			log.Error(err)
		}
		return
	}

	// 我们通过FindStringSubmatch 捕捉用户输入的命令，然后放入match切片，然后buyRegexp.SubexpNames() 遍历我们定义的正则表达式，通过SubexpNames() 方法，以切片形式返回，如果 name 不为空字符串，你将捕获组的内容（match[i]）存入一个映射 command 中，键是捕获组的名称（如 pair 或 amount），值是对应的匹配结果（如 BTCUSDT 或 100）。
	command := make(map[string]string)
	for i, name := range buyRegexp.SubexpNames() {
		if i != 0 && name != "" {
			command[name] = match[i]
		}
	}

	// command["pair"] 从之前创建的 command 映射中获取与键 "pair" 相关联的值。将用户输入的交易对名称转换成大写格式。在许多交易系统中，交易对的标准表示是大写字母，如 "BTCUSDT"。转换为大写可以确保程序在处理和比较字符串时的一致性和无误差。
	pair := strings.ToUpper(command["pair"])
	// 从用户输入的命令中通过键 "amount" 从映射 command 中提取字符串值，然后尝试将这个字符串转换为 float64 类型的数值，存放在变量 amount 中。
	amount, err := strconv.ParseFloat(command["amount"], 64)
	if err != nil {
		//如果 err 不为空（即存在错误），使用 log.Error(err) 记录错误日志
		log.Error(err)
		t.OnError(err)
		return
	} else if amount <= 0 {
		//如果用户输入的金额数量小于或者等于o 就返回一个错误，无效的金额
		_, err := t.client.Send(m.Sender, "Invalid amount")
		if err != nil {
			log.Error(err)
		}
		return
	}

	// 这一行检查 command 映射中是否包含 "percent" 键且该键对应的值不为空。值不为空意味着用户输入的命令中包括了一个百分比符号（%），如 /buy BTCUSDT 50% 中的 50%。
	if command["percent"] != "" {
		// 通过Position方法传入用户输入的交易对作为参数拿到报价资产
		_, quote, err := t.orderController.Position(pair)
		if err != nil {
			log.Error(err)
			t.OnError(err)
			return
		}
		// 如果用户输入 /buy BTCUSDT 100，则 amount 表示直接的金额；如果用户输入 /buy BTCUSDT 50%，则 amount 表示百分比
		// 计算实际使用的资金金额 = 百分比 x 报价资产数量 / 100
		// 例如，如果报价资产有100USDT，用户输入 /buy BTCUSDT 50%，实际使用的资金金额 = 50 * 100 / 100.0 = 50 USDT
		amount = amount * quote / 100.0
	}

	// 创建市场订单。
	order, err := t.orderController.CreateOrderMarketQuote(model.SideTypeBuy, pair, amount)
	if err != nil {
		return
	}
	// 记录购买订单的创建。
	log.Info("[TELEGRAM]: BUY ORDER CREATED: ", order)
}

// SellHandle 处理通过 Telegram 消息接收到的卖出命令。
func (t telegram) SellHandle(m *tb.Message) {
	// 使用正则表达式匹配用户的输入命令。
	match := sellRegexp.FindStringSubmatch(m.Text)
	// 如果没有匹配到任何内容，说明用户输入的命令格式不正确。
	if len(match) == 0 {
		_, err := t.client.Send(m.Sender, "Invalid command.\nExample of usage:\n`/sell BTCUSDT 100`\n\n`/sell BTCUSDT 50%`")
		if err != nil {
			// 如果发送错误信息失败，则记录错误。
			log.Error(err)
		}
		return
	}

	// 创建一个映射来存储命令的参数，从正则表达式的命名组中提取。
	command := make(map[string]string)
	for i, name := range sellRegexp.SubexpNames() {
		if i != 0 && name != "" {
			// 将匹配的结果按组名存入映射。
			command[name] = match[i]
		}
	}

	// 将交易对字符串转换为大写。
	pair := strings.ToUpper(command["pair"])
	// 尝试将字符串形式的数量转换为浮点数。
	amount, err := strconv.ParseFloat(command["amount"], 64)
	if err != nil {
		// 如果转换失败，则记录错误并调用错误处理函数，然后返回。
		log.Error(err)
		t.OnError(err)
		return
	} else if amount <= 0 {
		// 如果转换的数量小于或等于零，则发送无效数量的错误信息。
		_, err := t.client.Send(m.Sender, "Invalid amount")
		if err != nil {
			// 如果发送错误信息失败，则记录错误。
			log.Error(err)
		}
		return
	}

	// 检查是否有百分比指定。
	if command["percent"] != "" {
		// 如果指定了百分比，获取基础资产的头寸。
		asset, _, err := t.orderController.Position(pair)
		if err != nil {
			// 如果获取头寸失败，直接返回。
			return
		}
		// 计算基于头寸的指定百分比的金额。
		amount = amount * asset / 100.0
		// 创建一个市场卖出订单。
		order, err := t.orderController.CreateOrderMarket(model.SideTypeSell, pair, amount)
		if err != nil {
			// 如果创建订单失败，直接返回。
			return
		}
		// 记录订单创建信息。
		log.Info("[TELEGRAM]: SELL ORDER CREATED: ", order)
		return
	}

	// 创建一个常规的市场卖出订单。
	//就是为了满足两个用户以上同时输入不同的命令，一个输入百分比，一个输入数量，然后不同的命令对应着不同的市场订单，交易平台能够同时满足不同用户的需求，提供更加灵活和用户友好的交易体验，从而吸引更多的用户并提高平台的竞争力。
	order, err := t.orderController.CreateOrderMarketQuote(model.SideTypeSell, pair, amount)
	if err != nil {
		// 如果创建订单失败，直接返回。
		return
	}
	// 记录订单创建信息。
	log.Info("[TELEGRAM]: SELL ORDER CREATED: ", order)
}

// 函数用来处理 "/status" 命令。它首先通过 t.orderController.Status() 获取订单控制器的状态，然后将状态信息发送给消息的发送者，使用 Markdown 格式将状态信息包裹为粗体字体。
func (t telegram) StatusHandle(m *tb.Message) {
	// 获取订单控制器的状态
	status := t.orderController.Status()
	// 向消息发送者发送订单控制器的状态信息，使用 Markdown 格式包裹状态信息，以便显示为粗体
	//在代码中，状态信息被包裹在反引号（``）中，这是Markdown的一种语法，用于在文本中表示内联代码或强调。虽然它没有明确指定要将文本显示为粗体，但它确实应用了Markdown的一部分。
	_, err := t.client.Send(m.Sender, fmt.Sprintf("Status: `%s`", status))
	// 如果发送消息时出现错误，则记录错误
	if err != nil {
		log.Error(err)
	}
}

// 这个函数的主要目的是根据用户发送的 "/start" 命令来启动或通知机器人的运行状态，并向用户发送相应的消息。
func (t telegram) StartHandle(m *tb.Message) {
	// 检查订单控制器的状态是否正在运行
	if t.orderController.Status() == order.StatusRunning {
		// 如果订单机器人已经在运行，则向发送者发送消息通知“机器人已经在运行”，并附带默认菜单
		//因为在在 NewTelegram 中将设置好的菜单传递给了telegram结构体示例，所以t.defaultMenu就是设置好菜单
		_, err := t.client.Send(m.Sender, "Bot is already running.", t.defaultMenu)
		// 如果发送消息时出现错误，则记录错误并返回
		if err != nil {
			log.Error(err)
		}
		return
	}

	// 启动订单机器人
	t.orderController.Start()
	// 向发送者发送消息通知“机器人已经启动”，并附带默认菜单
	_, err := t.client.Send(m.Sender, "Bot started.", t.defaultMenu)
	// 如果发送消息时出现错误，则记录错误
	if err != nil {
		log.Error(err)
	}
}

// 当用户发送停止命令时，机器人会停止执行任何操作，并向用户发送消息确认机器人已经停止。
func (t telegram) StopHandle(m *tb.Message) {
	// 检查订单控制器的状态是否已经是已停止。
	if t.orderController.Status() == order.StatusStopped {
		// 如果订单控制器的状态已经是已停止，则向消息发送者发送消息 "Bot is already stopped."，并附带默认菜单。
		_, err := t.client.Send(m.Sender, "Bot is already stopped.", t.defaultMenu)
		// 如果发送消息时出现错误，记录错误日志。
		if err != nil {
			log.Error(err)
		}
		return
	}

	// 停止订单控制器的运行。
	t.orderController.Stop()
	// 向消息发送者发送消息 "Bot stopped."，并附带默认菜单。
	_, err := t.client.Send(m.Sender, "Bot stopped.", t.defaultMenu)
	// 如果发送消息时出现错误，记录错误日志。
	if err != nil {
		log.Error(err)
	}
}

func (t telegram) OnOrder(order model.Order) {
	// 根据订单状态选择合适的标题。
	title := ""
	switch order.Status {
	case model.OrderStatusTypeFilled:
		title = fmt.Sprintf("✅ ORDER FILLED - %s", order.Pair)
	case model.OrderStatusTypeNew:
		title = fmt.Sprintf("🆕 NEW ORDER - %s", order.Pair)
	case model.OrderStatusTypeCanceled, model.OrderStatusTypeRejected:
		title = fmt.Sprintf("❌ ORDER CANCELED / REJECTED - %s", order.Pair)
	}

	// 组装通知消息。
	message := fmt.Sprintf("%s\n-----\n%s", title, order)
	// 调用 Notify 方法发送通知消息。
	t.Notify(message)
}

// OnError 方法是用于处理和响应错误的一种通用处理函数，在 telegram 类型的实例中定义。这个方法主要用于当机器人运行过程中遇到错误时，对错误进行格式化显示并通过 Telegram 通知用户。
func (t telegram) OnError(err error) {
	// 设置错误信息的标题
	title := "🛑 ERROR"

	// 创建一个指向 exchange.OrderError 类型的指针
	var orderError *exchange.OrderError
	// 尝试将 err 接口类型转换为 *exchange.OrderError 类型，以检查这个错误是否是一个订单错误
	//errors.As 函数用来检查 err（一个错误对象）是否可以被视为或转换为指定的错误类型，这里是 *exchange.OrderError。该函数尝试将 err "视为" orderError 的类型，如果成功，orderError 变量会指向 err 的实际存储内容，这允许你访问特定类型的字段和方法。
	if errors.As(err, &orderError) {
		// 如果错误是订单错误，格式化错误信息，包括交易对、数量和具体错误信息
		message := fmt.Sprintf(`%s
        -----
        Pair: %s
        Quantity: %.4f
        -----
        %s`, title, orderError.Pair, orderError.Quantity, orderError.Err)
		// 使用 Notify 方法将格式化后的错误信息发送给用户
		t.Notify(message)
		return // 早退，因为错误已经被处理
	}

	// 如果错误不是订单错误，只简单地显示错误标题和错误信息
	t.Notify(fmt.Sprintf("%s\n-----\n%s", title, err))
}
