package order

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/storage"

	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
)

// summary结构体存储了特定交易对的统计信息。
type summary struct {
	Pair             string    // 交易对，例如"BTC/USD"
	WinLong          []float64 // 获利的多仓交易金额列表,对于那些买入某资产并随后价格上涨后卖出的成功交易，WinLong会记录每次交易所获得的盈利额放在一个切片里面
	WinLongPercent   []float64 // 获利的多仓交易百分比列表,如果你买入价值100美元的股票，并在价值上升到110美元时卖出，那么你的盈利是10美元，盈利百分比是10%。这个10%就是会被记录在WinLongPercent列表中的。如果你进行了多次这样的交易，每次的盈利百分比都会被依次记录在这个列表中
	WinShort         []float64 // 获利的空仓交易金额列表,例如，如果交易者借入并立即卖出股票，每股价格为100美元，后来价格下跌到90美元时买回，那么他们每股赚了10美元。如果他们卖出了10股，那么这次交易的盈利金额是100美元（10股 * 每股10美元），这个数字就会被添加到WinShort列表中。
	WinShortPercent  []float64 // 获利的空仓交易百分比列表,如果交易者做空一个资产，从中赚得了100美元，而他们最初卖出资产的价值是1000美元，那么盈利百分比就是 (100 / 1000) * 100 = 10%。这个10%就会被添加到WinShortPercent
	LoseLong         []float64 // 亏损的多仓交易金额列表,例如，如果一个交易者买入价值100美元的股票，希望价格会上涨，但股票价格下跌到90美元时他决定卖出，那么他在这次交易中的亏损就是10美元。如果这种亏损的交易发生了多次，每次的亏损金额就会被依次记录在LoseLong列表中。
	LoseLongPercent  []float64 // 亏损的多仓交易百分比列表,例如，如果一个交易者买入价值100美元的股票，但在未来股票价格下跌到90美元时卖出，造成10美元的亏损，那么这次交易的亏损百分比就是10%。
	LoseShort        []float64 // 亏损的空仓交易金额列表,如果一个交易者卖出价值100美元的股票，但随后股票价格上涨到110美元时被迫回购，造成10美元的亏损，那么这次交易的亏损金额就会被记录在LoseShort列表中。
	LoseShortPercent []float64 // 亏损的空仓交易百分比列表,例如，如果一个交易者以100美元的价格卖出了某资产，但在之后以110美元的价格回购该资产，造成了10%的亏损，那么这次交易的亏损百分比就会被记录在LoseShortPercent列表中。
	Volume           float64   // 在这个交易对上交易的总量,这个字段记录了在给定的交易对上所有交易的总量，无论是买入还是卖出。
}

// Win方法返回所有获利交易的金额列表，包括多仓和空仓。
func (s summary) Win() []float64 {
	return append(s.WinLong, s.WinShort...) // 将多仓和空仓的获利交易金额合并到一个列表,过将获利的多仓和空仓交易金额合并到一个列表中，可以简化对获利交易的处理逻辑，例如计算总获利、平均获利等指标时更加方便。
}

// WinPercent方法返回所有获利交易的百分比列表，包括多仓和空仓。
func (s summary) WinPercent() []float64 {
	return append(s.WinLongPercent, s.WinShortPercent...) // 将多仓和空仓的获利交易百分比合并到一个列表,例如计算总平均获利百分比、最大获利百分比等指标时更加方便。这样做也有助于减少代码重复和提高代码的可读性，使得代码更加简洁清晰。
}

// Lose方法返回所有亏损交易的金额列表，包括多仓和空仓。
func (s summary) Lose() []float64 {
	return append(s.LoseLong, s.LoseShort...) // 将多仓和空仓的亏损交易金额合并到一个列表,例如计算总亏损金额、平均亏损金额等指标时更加方便。
}

// LosePercent方法返回所有亏损交易的百分比列表，包括多仓和空仓。
func (s summary) LosePercent() []float64 {
	return append(s.LoseLongPercent, s.LoseShortPercent...) // 将多仓和空仓的亏损交易百分比合并到一个列,例如计算总亏损百分比、平均亏损百分比等指标时更加方便。表
}

// Profit 方法计算所有交易的总利润。
func (s summary) Profit() float64 {
	// 初始化利润为0
	profit := 0.0
	//总利润和总损失结合成一个数组之后，遍历之后得到一个单个的有可能是盈利，有可能是损失，然后相加，得到总利润
	// /假设赢得的交易利润：s.Win() 返回 [100, 200]，失去的交易亏损：s.Lose() 返回 [-50, -150]使用 append(s.Win(), s.Lose()...) 将这两个切片合并得到一个新的切片 [100, 200, -50, -150]，遍历之后依次得到100, 200, -50, -150 ，这里面的收益有可能是正的，或者负的，然后相加得到总收益加上第一笔交易的利润后，profit 变为 0 + 100 = 100，profit 变为 100 + 200 = 300。profit 变为 300 - 50 = 250。最后，加上第四笔交易的亏损后，profit 变为 250 - 150 = 100。
	for _, value := range append(s.Win(), s.Lose()...) {
		profit += value
	}
	// 返回总利润
	return profit
}

// SQN 方法计算系统的 SQN（System Quality Number）值。
// 这个SQN()方法通过计算交易数量、平均利润和利润的标准差，来评估交易系统的质量。SQN值越高，表示交易系统的性能越好，因为它意味着系统能够在较小的波动性下实现较高的平均利润。
func (s summary) SQN() float64 {
	// 行代码确实是用来获取总交易数，也就是总订单数的。它通过计算 s.Win() 返回的列表长度（盈利的交易数）和 s.Lose()
	total := float64(len(s.Win()) + len(s.Lose()))

	// 计算平均利润=总利润/总订单数
	avgProfit := s.Profit() / total

	// 初始化标准差为0，在交易系统分析中，标准差可以用来衡量交易利润的波动性。一个较低的标准差表示交易利润比较稳定，波动性小；而较高的标准差则表示交易利润波动性大，风险可能也更高。
	stdDev := 0.0

	// 计算利润的标准差
	// 例如一个交易系统五次交易的利润是￥100，￥50，￥0，-￥50，-￥100。
	// 第一步：计算平均利润 = 所有利润之和 / 总数 = (100 + 50 + 0 - 50 - 100) / 5 = 0
	// 第二步：计算方差。首先计算每个利润与平均利润之差的平方，然后将这些平方值相加。
	// 例如，对于￥100的利润，其差的平方是(100 - 0)^2 = 10000。
	// 第三步：求方差的平均值 = 所有差的平方之和 / 总交易数 = (10000 + 2500 + 0 + 2500 + 10000) / 5 = 4000
	// 第四步：标准差 = 方差的平方根 = √4000 ≈ 63.25
	for _, profit := range append(s.Win(), s.Lose()...) {
		//ath.Pow是指定次幂第一个是数字，第二个就是要求的次幂基数，例如math.Pow(2, 2)=2²=4
		// 通过循环将每个利润的差的平方累加，得到总的差的平方和，用于后续计算方差。
		stdDev += math.Pow(profit-avgProfit, 2)
	}

	// 标准差取平方根，完成标准差的计算
	//(标准差=平均方差/总数)然后再开平方  math.Sqrt(stdDev / total)  总方差/总数然后通过math.Sqrt开平方，得到标准差stdDev
	stdDev = math.Sqrt(stdDev / total)

	// 返回 SQN 值，根据公式 SQN = sqrt(N) * (avgProfit / stdDev) 计算
	//SQN = ✔总交易数 x (平均利润/标准差)
	return math.Sqrt(total) * (avgProfit / stdDev)
}

// 这个 Payoff 方法的代码主要目的是计算一个交易系统的回报率。
// 回报率是盈利交易的平均盈利除以亏损交易的平均亏损的绝对值。
// 如果没有盈利交易、亏损交易或平均亏损为0，则返回0。
func (s summary) Payoff() float64 {
	// 初始化平均盈利百分比和平均亏损百分比为0
	avgWin := 0.0
	avgLose := 0.0

	// 计算所有盈利交易的平均盈利
	// 遍历通过s.WinPercent()方法返回的盈利交易百分比列表
	for _, value := range s.WinPercent() {
		avgWin += value // 将每个盈利交易的百分比加到avgWin变量上
	}

	// 计算所有亏损交易的平均亏损
	// 遍历通过s.LosePercent()方法返回的亏损交易百分比列表
	for _, value := range s.LosePercent() {
		avgLose += value // 将每个亏损交易的百分比加到avgLose变量上
	}

	// 如果没有盈利交易、亏损交易或平均亏损为0，则返回0
	// 这是为了避免除以零的情况，并确保只有在有盈利和亏损交易时才计算回报率
	if len(s.Win()) == 0 || len(s.Lose()) == 0 || avgLose == 0 {
		return 0
	}

	// 计算回报率
	// 回报率是通过将盈利交易的平均盈利除以亏损交易的平均亏损的绝对值来计算的
	// 首先，将所有盈利百分比的总和除以盈利交易的数量得到平均盈利
	// 然后，将所有亏损百分比的总和除以亏损交易的数量得到平均亏损
	// 最后，计算平均盈利与平均亏损绝对值的比例，得到回报率
	// (avgWin / float64(len(s.Win()))) 得到平均每笔交易订单的盈利百分比，
	// math.Abs(avgLose/float64(len(s.Lose())))得到平均每单的损失百分比， math.Abs 函数确保这个值是正数（绝对值）
	//回报率 = 每单盈利交易的平均盈利百分比/每单亏损交易的平均亏损百分比的绝对值来计算得到的，回报率越高表示更强的盈利能力，更好的风险管理，更高的投资吸引力
	return (avgWin / float64(len(s.Win()))) / math.Abs(avgLose/float64(len(s.Lose())))
}

// ProfitFactor 计算总盈利与总亏损的比值。如果没有亏损，则返回0。
// 通过分析盈亏的百分比比值，交易者可以对其交易策略的风险敏感度有一个基本的了解。一个高的比值可能意味着策略在盈利方面表现较好，而一个低的比值可能表明亏损占据了上风。
func (s summary) ProfitFactor() float64 {
	// 如果亏损次数为0，则直接返回0，避免除以0的错误
	if len(s.Lose()) == 0 {
		return 0
	}
	// 初始化盈利总额
	profit := 0.0
	// 遍历赢利百分比，累加到盈利总额
	for _, value := range s.WinPercent() {
		profit += value
	}

	// 初始化亏损总额
	loss := 0.0
	// 遍历亏损百分比，累加到亏损总额
	for _, value := range s.LosePercent() {
		loss += value
	}
	// 返回盈利与亏损的绝对值比值
	return profit / math.Abs(loss)
}

// WinPercentage 这段代码的功能是计算赢利交易在所有交易中所占的比例，然后将这个比例转换为百分比形式。这个指标通常被称作胜率（Win Rate）或赢利交易的百分比
// 例如总交易数是10次盈利6次，亏损4次，6 / 10 x 100 = 60%
// 效率衡量：胜率提供了一个简单的衡量方法，帮助交易者快速了解自己的交易策略在市场上的表现。高胜率意味着在考察期间，赢利交易的比例较高。
func (s summary) WinPercentage() float64 {
	// 如果赢利和亏损交易的总次数为0，则返回0
	if len(s.Win())+len(s.Lose()) == 0 {
		return 0
	}
	// 计算赢利交易次数占总交易次数的比例，并乘以100转换为百分比
	return float64(len(s.Win())) / float64(len(s.Win())+len(s.Lose())) * 100
}

// String 生成并返回交易总结的表格字符串。
// 意思就是我现在创建一个tableString字符串构建器，然后再创建一个表格写入器，写入的目标是tableString字符串构建器，然后准备好数据，向表格批量添加数据，设置表格第一列左对齐，第二列右对齐，渲染表格，然后返回字符串构建器，因为表格的目标是写入构建器，所以构建器里面已经有表格了
func (s summary) String() string {
	// 初始化一个字符串构建器，用于构建表格字符串，它提供了一种高效的方式来创建字符串，因为它允许直接向一个缓冲区追加字符串，而不是在每次操作时都创建新的字符串实例。
	tableString := &strings.Builder{}
	// 创建一个新的表格写入器，ablewriter是一个流行的Go库，用于在ASCII格式下创建和管理表格。这个库提供了各种功能来定制表格的外观，比如设置边框、列宽、对齐方式等。通过指定tableString为输出目标，tablewriter生成的表格将会被写入到这个strings.Builder实例中，从而可以通过调用tableString.String()方法获取最终生成的表格字符串。
	table := tablewriter.NewWriter(tableString)
	// 从交易对中分离出报价货币
	_, quote := exchange.SplitAssetQuote(s.Pair)
	// 准备表格数据
	data := [][]string{
		{"Coin", s.Pair}, // 交易对
		{"Trades", strconv.Itoa(len(s.Lose()) + len(s.Win()))}, // 交易总次数
		{"Win", strconv.Itoa(len(s.Win()))},                    // 赢利次数
		{"Loss", strconv.Itoa(len(s.Lose()))},                  // 亏损次数
		{"% Win", fmt.Sprintf("%.1f", s.WinPercentage())},      // 赢利百分比
		{"Payoff", fmt.Sprintf("%.1f", s.Payoff()*100)},        // 支付比率（可能是代码中遗漏的方法）
		{"Pr.Fact", fmt.Sprintf("%.1f", s.Payoff()*100)},       // 盈亏比（此处可能有误，应该是盈亏因子）
		{"Profit", fmt.Sprintf("%.4f %s", s.Profit(), quote)},  // 总盈利
		{"Volume", fmt.Sprintf("%.4f %s", s.Volume, quote)},    // 交易量
	}
	//data是一个二维切片（[][]string），其中每个元素（一个[]string切片）代表表格的一行数据。
	//使用AppendBulk(data)方法，就可以一次性将多行数据添加到表格中，而不需要逐行添加。这对于处理大量数据时可以提高效率。
	// 向表格中批量添加数据
	table.AppendBulk(data)
	// 设置表格的列对齐方式，这行代码是用来设置表格中列的对齐方式。它是tablewriter库的一个功能，tablewriter是Go语言的一个库，用于在终端或ASCII文本中创建格式化的表格。这个特定的方法
	//SetColumnAlignment 接收一个切片参数，参数的每个数都代表一列，现在有两个数，所以代表两列，第一列左对齐，第二列右对齐
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
	// 渲染表格，并将其输出到字符串构建器中
	table.Render()
	// 返回构建的表格字符串
	return tableString.String()
}

// 这个SaveReturns方法是summary结构体的一个成员方法，其功能是将交易的盈利百分比和亏损百分比保存到一个指定的文件中
// 方法SaveReturns接收的是一个文件名（filename字符串）作为参数
func (s summary) SaveReturns(filename string) error {
	// 尝试创建一个名为filename的文件，用于数据写入。
	file, err := os.Create(filename)
	if err != nil {
		// 如果文件创建失败，返回错误。
		return err
	}
	// 使用defer关键字来确保在函数返回前关闭文件，释放资源。
	defer file.Close()

	// 遍历结构体中的WinPercent切片，将每个赢利百分比写入文件。
	for _, value := range s.WinPercent() {
		// 将每个百分比格式化为小数点后四位，并添加换行符，写入文件。
		_, err = file.WriteString(fmt.Sprintf("%.4f\n", value))
		if err != nil {
			// 如果写入过程中发生错误，返回错误。
			return err
		}
	}

	// 遍历结构体中的LosePercent切片，将每个亏损百分比写入文件。
	for _, value := range s.LosePercent() {
		// 同样将每个百分比格式化并写入文件。
		_, err = file.WriteString(fmt.Sprintf("%.4f\n", value))
		if err != nil {
			// 如果写入过程中发生错误，返回错误。
			return err
		}
	}
	// 如果所有数据都成功写入，返回nil表示成功。
	return nil
}

// Status类型用于描述一个过程或任务的当前状态。
// 先定义一个状态类型，然后在他身上绑定不同的变量，这样易于管理将所有可能的状态集中在一个地方定义，使得管理和更新状态变得更加简单，避免错误：当你为状态定义一个专门的类型时，这个类型的变量就只能接受预定义的状态值。高代码可读性：使用明确的状态名称（如StatusRunning、StatusStopped等）而不是裸露的字符串或数字，可以让其他开发者（或未来的你）更容易理解代码的意图。
type Status string

// 定义Status类型可能的值：running、stopped和error。
const (
	StatusRunning Status = "running" // 表示正在进行中。
	StatusStopped Status = "stopped" // 表示已停止。
	StatusError   Status = "error"   // 表示发生错误。
)

// Result结构体用于存储交易的结果数据。
type Result struct {
	Pair          string         // 交易对。
	ProfitPercent float64        // 盈利百分比。
	ProfitValue   float64        // 盈利金额。
	Side          model.SideType // 交易方向（买入/卖出）。
	Duration      time.Duration  // 交易持续时间。
	CreatedAt     time.Time      // 结果创建时间。
}

// Position结构体用于描述一个交易头寸的详细信息。交易头寸的意思就是，交易的方向，数量，价格，时间
type Position struct {
	Side      model.SideType // 头寸方向（买入/卖出）。
	AvgPrice  float64        // 平均价格。
	Quantity  float64        // 数量。
	CreatedAt time.Time      // 头寸创建时间。
}

// Update方法接收一个指向Order的指针作为参数，返回一个指向Result的指针和一个布尔值finished，表示头寸是否已结束。
//
// 现在买入2个BTC ，2个BTC就是我的头寸，当我出售一个后，还剩下一个，我的头寸还剩下1个BTC ，头寸里面有买卖方向：我是通过做多还是做空，买入这个BTC的这个例子是通过做多来的，数量：几个BTC，创建这个头寸的时间，就是买入这2个btc的时间
//
// 订单的意思就是我通过买入或者卖出这个资产，通过指令，买入的话呢就会创造一个新的头寸资产，卖出的话呢就会减少，
func (p *Position) Update(order *model.Order) (result *Result, finished bool) {
	// 通常情况下，交易的价格是订单的价格。
	price := order.Price

	// 如果订单类型是止损或止损限价，使用订单的止损价格作为交易价格。
	//止损加就是用来规避风险的，所以如果订单的类型市止损单，或者值止损市单，就把订单类型设置成为止损价，这样能够更好的止损，规避风险
	if order.Type == model.OrderTypeStopLoss || order.Type == model.OrderTypeStopLossLimit {
		price = *order.Stop
	}

	// 如果订单的方向与头寸的方向相同（即都是买入或都是卖出）。
	if p.Side == order.Side {
		// 更新平均价格，计算新的平均价格，考虑到新订单的价格和数量。
		//新的平均价格=(现有头寸的平均价格 x 现有数量  +  新订单的价格 x 新订单数量) / 现有数量 + 新订单数量
		p.AvgPrice = (p.AvgPrice*p.Quantity + price*order.Quantity) / (p.Quantity + order.Quantity)
		// 更新头寸的数量，加上新订单的数量。
		p.Quantity += order.Quantity
	} else {
		// 如果订单的方向与头寸方向相反，处理平仓和反向开仓的情况。
		if p.Quantity == order.Quantity {
			// 如果数量相等，表示完全平仓，头寸结束。
			finished = true
		} else if p.Quantity > order.Quantity {
			// 如果头寸的数量大于订单的数量，减少头寸的数量。
			p.Quantity -= order.Quantity
		} else {
			// 如果订单的数量大于头寸的数量，表示反向开仓。
			//如果你的原有头寸是空头（即你之前卖出了你没有的资产，预期价格下跌），然后你执行了一个买入订单，且这个买入的数量大于你的空头头寸数量这个买入操作首先会用相等的数量平仓（关闭）你的空头头寸，即“还回”你之前借入并卖出的资产。如果买入的数量超过了你空头头寸的数量，那么超出的部分将创建一个新的多头头寸。这表示，你不仅归还了借入的资产，而且还买入了更多的资产，现在实际持有它们，预期价格上涨。
			//反过来，如果你的原有头寸是多头（即你之前买入了资产，预期价格上涨），然后你执行了一个卖出订单，且卖出的数量大于你的多头头寸数量：这个卖出操作首先会平仓你的多头头寸如果卖出的数量超过了你多头头寸的数量，那么超出的部分将创建一个新的空头头寸。这表示，你不仅卖出了你持有的所有资产，而且还卖出了更多你目前不持有的资产，现在实际上是预期价格下跌。
			p.Quantity = order.Quantity - p.Quantity
			// 更新头寸的方向、创建时间和平均价格为新订单的相应值。
			p.Side = order.Side
			p.CreatedAt = order.CreatedAt
			p.AvgPrice = price
		}

		// 计算实际盈亏的数量，取两者数量的最小值。
		// 假如我有100BTC，如果要卖50 order.Quantity = 50 ，选择最小的交易，还剩下100，如果卖150  order.Quantity = 150 ，我只能交易100BTC 因为我只有这么多， 意思就是想要卖出的订单，必要在我合理的头寸数量范围内
		quantity := math.Min(p.Quantity, order.Quantity)
		// 计算盈亏比例。当前的订单价格 - 平均购买头寸的价格 )/平均购买头寸的价格 = 盈亏比例 如果比例 为正，意思就是获得了盈利， 为负的话就是亏损了
		//假设您的平均购买价格是8000美元，现在的订单价格是9000美元 (9000-8000)/8000 = 0.125 转换为百分比，即12.5%的盈利。
		order.Profit = (price - p.AvgPrice) / p.AvgPrice
		// 盈亏金额=(当前订单价格−平均购买头寸的价格)×交易数量
		// 假设您持有100BTC的头寸，平均购买价格是$8000/BTC。现在，价格上涨到$9000/BTC，您打算卖出50BTC。
		// (9000-8000) x 50 = 1000×50=$50,000 你的盈利是$50,000。
		order.ProfitValue = (price - p.AvgPrice) * quantity

		// 创建一个Result对象，记录交易结果。
		result = &Result{
			CreatedAt: order.CreatedAt,
			Pair:      order.Pair,
			//这段代码记录的是从买入（创建）这个头寸开始，到创建订单将其卖出为止的时间长度。
			//头寸创建于1月1日，当前订单创建于1月10日。会计算出两个日期之间的差值，即9天。这意味着从创建头寸到执行这个订单，总共经过了9天的时间。
			Duration:      order.CreatedAt.Sub(p.CreatedAt), // 计算持仓时长。
			ProfitPercent: order.Profit,
			ProfitValue:   order.ProfitValue,
			Side:          p.Side,
		}

		// 返回交易结果和头寸是否结束的标志。
		//果头寸被完全平仓或通过订单被反向开仓（即头寸方向改变，并且新的头寸量小于等于订单量），那么这个值会被设置为true，表示头寸已经结束，不再持有任何资产。如果头寸没有被完全平仓，这个值会是false，表示头寸仍然存在。
		return result, finished
	}

	// 如果订单的方向与头寸方向相同，不需要返回特殊的交易结果，返回nil和false。
	//订单的方向（买入或卖出）与头寸的方向相同时，这意味着该操作是在增加现有头寸的量（如果是买入操作）或减少但不完全平仓（如果是卖出操作），而不是在关闭或反向开仓。在这种情况下，返回nil和false
	//返回nil**代表没有特殊的交易结果需要记录或处理。这是因为头寸本质上没有发生质的改变，只是数量上的增减
	return nil, false
}

// Controller结构体将多个组件和服务整合在一起，管理交易逻辑的执行流程，包括交易操作、数据存储、实时数据订阅和通知发送等功能，形成了一个交易系统的核心部分。
// 这个控制器相当于一个交易机器人
type Controller struct {
	mtx            sync.Mutex          // 互斥锁，用于确保Controller操作的线程安全。mtx可以用来保证一次只有一个线程能够执行修改操作，防止数据竞争
	ctx            context.Context     // 上下文，用于控制长时间运行的操作，如取消操作等。
	exchange       service.Exchange    // 交易所接口，负责实际的交易操作。Controller 通过这个接口与外部的交易所进行交互，执行买入、卖出这类的交易操作
	storage        storage.Storage     // 存储接口，用于数据持久化。这可以是本地磁盘存储、数据库或者其他形式的解决方案，用来保存和检索交易数据，头寸信息
	orderFeed      *Feed               // 订单订阅源，用于接收订单相关的数据流。Controller 通过它接收外部订单和市场数据的更新
	notifier       service.Notifier    // 通知器接口，用于发送交易或头寸变动的通知。用于当交易或头寸发生变动时向用户或其他系统组件发生通知。这可以是邮件、短信、推送通知等形式
	Results        map[string]*summary // 存储每个标的的交易结果汇总。包含了盈亏信息、交易次数等统计数据 键是交易对
	lastPrice      map[string]float64  // 存储每个标的的最新价格。
	tickerInterval time.Duration       // 定时器间隔，用于定期执行某些操作。Control可以用它来定期执行一些操作，比如定时刷新市场数据、检查交易条件
	finish         chan bool           // 控制结束信号的通道，用于通知系统停止运行。用于通知Controller 可以在系统终止或者用户手动停止时发生信号到这个通道
	status         Status              // 控制器的当前状态。

	position map[string]*Position // 存储每个标的的当前头寸信息。 是一个映射，用于存储所有活跃的头寸信息，键是交易对的标识，值是对应的Position对象。
}

// NewController 是Controller的构造函数，用于初始化一个Controller实例。
func NewController(ctx context.Context, exchange service.Exchange, storage storage.Storage,
	orderFeed *Feed) *Controller {

	return &Controller{
		ctx:            ctx,
		storage:        storage,
		exchange:       exchange,
		orderFeed:      orderFeed,
		lastPrice:      make(map[string]float64),
		Results:        make(map[string]*summary),
		tickerInterval: time.Second, //表示定时器触发的间隔时间，默认设置为1秒。
		finish:         make(chan bool),
		position:       make(map[string]*Position),
	}
}

// SetNotifier 方法用于设置Controller的通知器组件，允许Controller在需要时发送通知。
func (c *Controller) SetNotifier(notifier service.Notifier) {
	c.notifier = notifier
}

// OnCandle 更新接收到的蜡烛图数据的最新收盘价。
func (c *Controller) OnCandle(candle model.Candle) {
	// 更新指定交易对的最新收盘价。
	c.lastPrice[candle.Pair] = candle.Close
}

// updatePosition 根据新订单信息更新或创建头寸。
func (c *Controller) updatePosition(o *model.Order) {
	// 尝试获取指定交易对的当前头寸。
	position, ok := c.position[o.Pair]
	if !ok {
		// 如果头寸不存在，则创建一个新的头寸并初始化它的基本信息。
		c.position[o.Pair] = &Position{
			AvgPrice:  o.Price,     // 设置头寸的平均价格为订单价格。
			Quantity:  o.Quantity,  // 设置头寸的数量为订单数量。
			CreatedAt: o.CreatedAt, // 记录头寸的创建时间。
			Side:      o.Side,      // 设置头寸的方向（买或卖）。
		}
		return // 头寸创建后直接返回。
	}

	// 如果头寸存在，使用新订单信息更新头寸，并检查是否已平仓。
	result, closed := position.Update(o)
	if closed {
		// 如果头寸已平仓，从头寸列表中删除该头寸。它从c.position映射中移除了对应的交易对（o.Pair）条目。
		delete(c.position, o.Pair)
	}

	// 如果有更新结果，根据结果的盈亏情况进行处理。
	//"有更新结果"意味着经过头寸更新操作后，存在一个交易结果。这个结果通常包含了交易的具体细节，如是否盈利或亏损、盈亏的金额、交易的方向（买或卖）等信息。
	if result != nil {
		// 根据盈亏百分比和买卖方向，更新统计数据。
		// 分为盈利和亏损两种情况，进一步分为买入和卖出两种交易方向。
		// 根据不同情况，将盈亏值和百分比分别追加到对应的统计列表中。

		// 根据订单信息，从交易对中分离出报价币种。
		_, quote := exchange.SplitAssetQuote(o.Pair)
		// 发送盈利通知，包括盈亏金额、报价币种、盈亏百分比以及统计信息的字符串表示。
		/*
					类似于：
					[PROFIT] 100.00 USD (5 %)
			          `交易结果`
		*/
		c.notify(fmt.Sprintf(
			"[PROFIT] %f %s (%f %%)\n`%s`",
			result.ProfitValue,
			quote,
			result.ProfitPercent*100,
			c.Results[o.Pair].String(), //这个交易对的交易结果
		))
	}
}

// notify 是一个负责发送通知消息的方法。
func (c *Controller) notify(message string) {
	// 首先，通过日志系统记录传入的消息。这可以帮助开发者在查看日志文件时了解系统状态和重要事件。
	log.Info(message)

	// 然后，检查是否配置了notifier（一个用于发送通知的组件）。这段代码检查notifier是否已经被实例化和设置。如果已经设置，那么会使用这个notifier来发送通知消息。这可以是发送电子邮件、短信、推送通知等，取决于notifier的具体实现。这主要是面向最终用户或者需要实时通知的场景。
	if c.notifier != nil {
		// 只要 Controller 的 notify 方法被调用，并且 Controller 实例中有一个已经被设置（即非 nil）的 notifier，就使用这个 notifier 发送通知消息给用户或系统
		// 这允许将消息发送到不同的目标，如电子邮件、短信、推送通知等，具体取决于notifier的实现。
		c.notifier.Notify(message)
	}
}

// notifyError 是一个方法，它接收一个错误作为输入并将其记录下来。如果设置了notifier（通知器），它还会将错误发送给notifier。
// 在开发交易机器人这样的系统时，如果系统运行过程中遇到了错误，通过实现一个notifier组件并利用notifier.OnError(err)这个接口，系统可以将错误信息通知给开发者或者系统管理员。
func (c *Controller) notifyError(err error) {
	log.Error(err)         // 使用日志系统记录错误信息
	if c.notifier != nil { // 检查是否配置了通知器
		c.notifier.OnError(err) // 如果配置了通知器，调用其OnError方法发送错误通知
	}
}

// processTrade 是一个方法，用于处理交易订单。它检查订单的状态，如果订单已经完成，则记录交易量并更新头寸大小和平均价格。
func (c *Controller) processTrade(order *model.Order) {
	if order.Status != model.OrderStatusTypeFilled { // 如果订单状态不是已完成，直接返回
		return
	}

	// 如果需要，初始化结果映射
	//看是否已经有了关于当前订单货币对的记录。如果没有（即ok为false），则为这个货币对初始化一个新的summary结构体实例，并将其添加到Results映射中。这保证了对每个货币对的操作都有一个对应的记录存在。
	if _, ok := c.Results[order.Pair]; !ok { // 检查指定货币对的结果是否已经初始化
		c.Results[order.Pair] = &summary{Pair: order.Pair} // 如果没有，初始化它
	}

	// 注册订单成交量
	//将订单的价格（order.Price）乘以订单的数量（order.Quantity），然后加到对应货币对的成交量（Volume）上。这样做是为了累计该货币对在所有“已完成”的订单中的总成交量。
	c.Results[order.Pair].Volume += order.Price * order.Quantity // 更新该货币对的成交量

	// 更新头寸大小/平均价格
	c.updatePosition(order) // 调用updatePosition方法更新头寸
}

// updateOrders 是一个方法，用于更新 Controller 中所有待处理的订单。
func (c *Controller) updateOrders() {
	c.mtx.Lock()         // 锁定互斥锁，确保同时只有一个线程可以执行更新操作。
	defer c.mtx.Unlock() // 函数结束时自动解锁。

	// 获取所有处于待处理状态的订单
	//把orders遍历完毕的单个订单order通过过滤器WithStatusIn方法厘米的制定了三个状态进行筛选，把数据库中的订单列表符合条件赛选出来，过滤条件是当订单列表里面的订单状态与WithStatusIn过滤器接收的状态相等就返回true这个订单就通过，应该保留，不符合就false ，表示该订单应该被排除。
	orders, err := c.storage.Orders(storage.WithStatusIn(
		model.OrderStatusTypeNew,             // 新订单
		model.OrderStatusTypePartiallyFilled, // 部分成交的订单
		model.OrderStatusTypePendingCancel,   // 等待取消的订单
	))
	if err != nil {
		c.notifyError(err) // 如果查询订单时出错，则发送错误通知
		return             // 并提前退出方法
	}

	// 遍历待处理的订单，检查它们是否有更新
	var updatedOrders []model.Order // 用于存储已更新订单的状态切片
	// 遍历待更新订单
	for _, order := range orders {
		excOrder, err := c.exchange.Order(order.Pair, order.ExchangeID) // 从交易所查询订单的状态
		if err != nil {
			//  就是向日志里面添加一个字段"id",内容为order.ExchangeID 交易所id，日志消息的前缀是 "orderControler/get: "，错误信息是由 err 变量提供的。 例如交易所id 123456，time="2024-04-08T12:00:00Z" level=error msg="orderControler/get: Failed to retrieve order details from the exchange" id=123456
			log.WithField("id", order.ExchangeID).Error("orderControler/get: ", err) // 记录查询失败的日志
			continue                                                                 // 跳过当前订单，处理下一个订单
		}

		// 检查交易所返回的订单状态是否与数据库中存储的订单状态相同。如果相同，则说明订单状态没有发生变化，无需进行更新，直接跳过处理下一个订单。
		if excOrder.Status == order.Status {
			continue
		}
		//这行代码确实将交易所返回的订单的ID（excOrder.ID）设置为与数据库中对应订单的ID（order.ID）相同。这确保了在数据库中正确识别需要更新的订单。
		excOrder.ID = order.ID                 // 确保更新后的订单有正确的内部ID
		err = c.storage.UpdateOrder(&excOrder) // 将从交易所返回的订单更新后，保存到数据库中
		if err != nil {
			c.notifyError(err) // 如果更新订单失败，发送错误通知
			continue           // 并继续处理下一个订单
		}

		log.Infof("[ORDER %s] %s", excOrder.Status, excOrder) // 记录订单状态更新的日志
		updatedOrders = append(updatedOrders, excOrder)       // 将更新后的订单添加到切片中
	}

	// 处理所有更新后的订单
	for _, processOrder := range updatedOrders {
		c.processTrade(&processOrder) // 处理交易逻辑
		//频道就是把一个整体的大项目分成很多小部分，分别分给不同的频道不同的人去完成，一旦这些小项目完成了，再合起来项目就可以进入下个阶段了,通过把订单更新事件发送到特定的频道，系统的其他部分（比如订单处理逻辑）就可以监听这个频道，一旦有更新事件发生，它们就进行相应的处理 。 相互频道独立完成工作的同时，又能协同完成整体目标 ，很好的实现了并发性
		c.orderFeed.Publish(processOrder, false) // 发布订单更新事件
	}
}

// Status 返回控制器当前的运行状态。
func (c *Controller) Status() Status {
	return c.status
}

// Start 方法启动控制器。如果控制器当前不处于运行状态，它将设置控制器状态为运行中，
// 并启动一个新的协程来周期性地更新订单。
// 归结起来，相当于启动了一个交易机器人。这个方法的执行流程确保了交易机器人只在未运行状态下启动，以防重复启动
func (c *Controller) Start() {
	// 如果控制器不处于运行状态，设置成运行状态
	if c.status != StatusRunning {
		// 设置控制器状态为运行中
		c.status = StatusRunning

		// 启动新的协程
		//这个协程能使控制器定期更新订单，这个更新不会受到主线程影响
		go func() {
			// 创建一个定时器，根据结构体c.tickerInterval参数来触发时间间隔
			//定时器 (time.Ticker) 到达设定的时间间隔时，就会向他的通道发送时间，这个就相当于信号，然后开始更新订单，stopChan 控制停止信号你可以从程序的任何地方（主goroutine或其他goroutine）发送一个信号到这个通道，以通知接收方（在这个例子中是监听 stopChan 的goroutine）停止执行，如stopChan <- true，然后就终止
			ticker := time.NewTicker(c.tickerInterval)
			for {
				select {
				// 当定时器的通道接收到信号时，调用updateOrders方法更新订单
				case <-ticker.C:
					c.updateOrders()
				// 如果收到停止信号，停止定时器并退出协程
				case <-c.finish:
					ticker.Stop()
					return
				}
			}
		}()
		// 记录启动日志
		log.Info("Bot started.")
	}
}

// Stop 方法停止控制器。如果控制器正在运行，它将设置控制器状态为已停止，
// 发送一个信号到finish通道来停止更新订单的协程，然后记录停止日志。
// 这个 Stop 方法的目的是安全地停止交易机器人的运行，并记录相关的停止信息。
func (c *Controller) Stop() {
	// 检查控制器是否正在运行
	if c.status == StatusRunning {
		// 设置控制器状态为已停止
		c.status = StatusStopped
		// 更新一次订单，可能是为了处理停止前的最终状态
		//这可能是为了确保在停止机器人之前，处理或保存其最终状态，比如关闭所有未完成的交易或更新交易策略的最终结果。
		c.updateOrders()
		// 发送停止信号到finish通道
		c.finish <- true
		// 记录停止日志
		log.Info("Bot stopped.")
	}
}

// Account 返回与当前控制器(交易机器人)关联的交易账户信息。
// 它调用交易所接口的 Account 方法来获取账户的模型数据。
// 调用交易所的接口，返回账户信息
func (c *Controller) Account() (model.Account, error) {
	return c.exchange.Account()
}

// 调用交易所Position接口，返回基础资产头寸，还有报价资产头寸
func (c *Controller) Position(pair string) (asset, quote float64, err error) {
	return c.exchange.Position(pair)
}

// 调用交易所LastQuote 接口返回最新报价
func (c *Controller) LastQuote(pair string) (float64, error) {
	return c.exchange.LastQuote(c.ctx, pair)
}

// PositionValue 计算并返回指定交易对当前头寸的价值。
// 方法首先通过调用exchange接口的Position方法来获取交易对的头寸信息。
func (c *Controller) PositionValue(pair string) (float64, error) {
	asset, _, err := c.exchange.Position(pair)
	if err != nil {
		return 0, err
	}
	//如果成功获取头寸信息，它会使用头寸中的资产数量（asset）乘以该交易对的最新价格（c.lastPrice[pair]）计算头寸价值。
	return asset * c.lastPrice[pair], nil
}

// Order 获取并返回指定交易对和订单ID的订单信息。
func (c *Controller) Order(pair string, id int64) (model.Order, error) {
	return c.exchange.Order(pair, id)
}

// Controller 结构体负责创建OCO订单，并处理与订单相关的逻辑，例如加锁以避免并发问题、记录日志、错误处理和订单数据的存储与发布。 size订单数量，price float64: 目标价格， stop 止损价，stopLimit止损限价
// （一单成交即取消另一单）确实是这样工作的。在金融交易中，OCO（One Cancels the Other）订单包括两个订单：一个止盈单和一个止损单。这两个订单同时下达，但是一旦其中一个条件被触发并且订单成交，另一个订单将自动被取消。
func (c *Controller) CreateOrderOCO(side model.SideType, pair string, size, price, stop, stopLimit float64) ([]model.Order, error) {
	c.mtx.Lock()         // 加锁，确保同时只有一个操作可以修改控制器的状态
	defer c.mtx.Unlock() // 函数执行结束时解锁，无论是正常结束还是由于错误提前返回

	log.Infof("[ORDER] Creating OCO order for %s", pair) // 记录日志，表示正在为指定交易对创建OCO订单

	// 调用exchange的CreateOrderOCO方法创建OCO订单
	// 此处传递订单参数：方向（买/卖）、交易对、订单大小、价格、止损价格和止损限价
	orders, err := c.exchange.CreateOrderOCO(side, pair, size, price, stop, stopLimit)
	if err != nil {
		c.notifyError(err) // 如果创建订单时出现错误，调用notifyError方法通知错误
		return nil, err    // 返回错误，中断函数执行
	}

	// 遍历创建的订单
	for i := range orders {
		//每个订单都被保存到数据库中，这样做的目的是为了记录订单的详细信息，以便于后续的查询、分析或是进行交易管理。
		err := c.storage.CreateOrder(&orders[i])
		if err != nil {
			c.notifyError(err) // 如果保存订单时出现错误，通知错误并返回
			return nil, err
		}
		//启动一个协程 把订单放到一个频道中，接收方（监听这个频道的其他部分或协程）将逐一处理这些订单。这个处理可能涉及到更新数据库、执行交易逻辑、发送通知等操作。
		go c.orderFeed.Publish(orders[i], true) // 异步地将订单信息发布到订单信息流中
	}

	return orders, nil // 返回创建的订单列表和nil错误，表示成功
}

// CreateOrderLimit 方法在 Controller 结构体中用于创建一个限价订单 limit限价
func (c *Controller) CreateOrderLimit(side model.SideType, pair string, size, limit float64) (model.Order, error) {
	c.mtx.Lock()         // 加锁，防止同时对Controller对象的并发修改
	defer c.mtx.Unlock() // 确保在函数退出时释放锁，无论是通过正常返回还是因为错误提前退出

	// 使用日志记录正在创建限价订单的信息，包括订单的方向（买/卖）、交易对
	log.Infof("[ORDER] Creating LIMIT %s order for %s", side, pair)

	// 调用交易所接口创建限价订单，传入订单方向、交易对、数量和限价
	order, err := c.exchange.CreateOrderLimit(side, pair, size, limit)
	if err != nil {
		// 如果创建订单过程中发生错误，记录并通知错误，然后返回一个空的Order对象和错误信息
		c.notifyError(err)
		return model.Order{}, err
	}

	// 将成功创建的订单保存到存储系统中，可能是数据库或其他形式的持久化存储
	err = c.storage.CreateOrder(&order)
	if err != nil {
		// 如果保存订单时发生错误，记录并通知错误，然后返回一个空的Order对象和错误信息
		c.notifyError(err)
		return model.Order{}, err
	}

	// 异步地将订单信息发布到订单信息流中，不会阻塞当前的操作
	//订单就被发送到这个信息流的Data频道中。这意味着该订单将被推送到订阅了这个特定交易对信息流的任何组件或服务中，这些组件或服务随后可以处理这个订单，比如执行交易、更新数据库或者触发其他业务逻辑。
	go c.orderFeed.Publish(order, true)

	// 记录订单创建成功的信息
	log.Infof("[ORDER CREATED] %s", order)

	// 返回创建的订单和nil错误，表示成功
	return order, nil
}

// CreateOrderMarketQuote 方法的目的是在交易系统中创建一个基于市场报价的订单amount 金额
func (c *Controller) CreateOrderMarketQuote(side model.SideType, pair string, amount float64) (model.Order, error) {
	c.mtx.Lock()         // 加锁以保证在创建订单过程中的线程安全
	defer c.mtx.Unlock() // 使用defer确保函数结束时解锁，即使是在返回错误时也能确保锁被释放

	// 记录日志，表示正在创建市价订单，包括订单的方向和交易对
	log.Infof("[ORDER] Creating MARKET %s order for %s", side, pair)

	// 调用交易所接口创建市价订单，传入订单的方向、交易对和金额
	order, err := c.exchange.CreateOrderMarketQuote(side, pair, amount)
	if err != nil {
		// 如果在创建订单时出现错误，通过notifyError方法记录并通知错误，然后返回空订单和错误信息
		c.notifyError(err)
		return model.Order{}, err
	}

	// 将新创建的订单保存到存储系统中，可能是数据库或其他形式的持久化存储
	err = c.storage.CreateOrder(&order)
	if err != nil {
		// 如果保存订单时出现错误，同样记录并通知错误，返回空订单和错误信息
		c.notifyError(err)
		return model.Order{}, err
	}

	// 计算交易产生的利润，processTrade可能包括了更新订单状态、计算利润等逻辑
	//因为市价订单立即以当前市场上可用的最佳价格执行，所以需要立即计算利润，立即更新数量，平均头寸，所以才有这个行代码
	c.processTrade(&order)

	// 异步地将订单信息发布到订单信息流中，不会阻塞当前的操作
	go c.orderFeed.Publish(order, true)

	// 记录订单创建成功的日志
	log.Infof("[ORDER CREATED] %s", order)

	// 返回创建的订单和可能的错误（如果之前的步骤没有错误，这里的err将是nil）
	return order, err
}

// CreateOrderMarket 方法的作用是在交易系统中创建一个市价订单
func (c *Controller) CreateOrderMarket(side model.SideType, pair string, size float64) (model.Order, error) {
	c.mtx.Lock()         // 在操作开始时加锁，以保证并发操作的线程安全
	defer c.mtx.Unlock() // 确保在函数结束时释放锁，无论函数是正常结束还是由于中途返回错误

	// 记录日志，表示正在创建市价订单，这里会显示订单的方向（买入/卖出）和交易对
	log.Infof("[ORDER] Creating MARKET %s order for %s", side, pair)

	// 调用与交易所交互的接口来创建市价订单，传入订单方向、交易对和订单大小
	order, err := c.exchange.CreateOrderMarket(side, pair, size)
	if err != nil {
		// 如果在创建订单的过程中出现错误，使用notifyError方法记录和通知错误
		c.notifyError(err)
		return model.Order{}, err // 返回一个空的订单对象和错误信息
	}

	// 将新创建的订单保存到存储系统中，这可能涉及数据库操作
	err = c.storage.CreateOrder(&order)
	if err != nil {
		// 如果保存订单时出现错误，同样记录和通知错误
		c.notifyError(err)
		return model.Order{}, err // 返回空订单和错误信息
	}

	// 计算交易产生的利润，具体实现可能包括更新订单的利润信息等
	//因为市价订单立即以当前市场上可用的最佳价格执行，所以需要立即计算利润，立即更新数量，平均头寸，所以才有这个行代码
	c.processTrade(&order)

	// 异步地将订单信息发布到订单信息流中，使用go关键字启动新的协程，以免阻塞当前操作
	go c.orderFeed.Publish(order, true)

	// 记录订单创建成功的信息
	log.Infof("[ORDER CREATED] %s", order)

	// 返回创建的订单对象和错误信息（如果之前的步骤没有出错，这里的err应该是nil）
	return order, err
}

// CreateOrderStop 方法在 Controller 结构体中用于创建一个止损订单（Stop Order）
func (c *Controller) CreateOrderStop(pair string, size float64, limit float64) (model.Order, error) {
	c.mtx.Lock()         // 在操作开始时加锁，以保证并发操作的线程安全
	defer c.mtx.Unlock() // 使用 defer 确保在函数结束时释放锁，无论函数是正常结束还是由于中途返回错误

	// 记录日志，表示正在为特定的货币对创建止损订单
	log.Infof("[ORDER] Creating STOP order for %s", pair)

	// 调用与交易所交互的接口来创建止损订单，传入货币对、订单大小和止损价格
	order, err := c.exchange.CreateOrderStop(pair, size, limit)
	if err != nil {
		// 如果在创建订单的过程中出现错误，使用 notifyError 方法记录和通知错误
		c.notifyError(err)
		return model.Order{}, err // 返回一个空的订单对象和错误信息
	}

	// 将新创建的订单保存到存储系统中，这可能涉及数据库操作
	err = c.storage.CreateOrder(&order)
	if err != nil {
		// 如果保存订单时出现错误，同样记录和通知错误
		c.notifyError(err)
		return model.Order{}, err // 返回空订单和错误信息
	}

	// 异步地将订单信息发布到订单信息流中，使用 go 关键字启动新的协程，以免阻塞当前操作
	go c.orderFeed.Publish(order, true)

	// 记录订单创建成功的信息
	log.Infof("[ORDER CREATED] %s", order)

	// 返回创建的订单对象和错误信息（如果之前的步骤没有出错，这里的 err 应该是 nil）
	return order, nil
}

// 这个 Cancel 方法是 Controller 结构体中用于取消一个订单的函数
func (c *Controller) Cancel(order model.Order) error {
	c.mtx.Lock()         // 在开始操作之前加锁，以确保并发操作时的数据一致性和线程安全
	defer c.mtx.Unlock() // 使用 defer 关键字来确保在函数退出时释放锁，无论是正常退出还是因为中途发生错误

	// 记录日志，指示正在取消特定货币对的订单
	log.Infof("[ORDER] Cancelling order for %s", order.Pair)

	// 调用交易所的接口尝试取消订单
	err := c.exchange.Cancel(order)
	if err != nil {
		// 如果取消订单的过程中发生错误，直接返回这个错误
		return err
	}

	// 将订单状态更新为待取消（PendingCancel）
	order.Status = model.OrderStatusTypePendingCancel

	// 更新存储系统中的订单信息，以反映其最新状态
	//一旦在交易所成功执行了取消订单的操作，同步更新本地数据库中的订单状态不仅有助于保持数据的一致性和准确性，而且对于交易决策、风险管理和用户体验等方面都是非常重要的。
	err = c.storage.UpdateOrder(&order)
	if err != nil {
		// 如果更新订单时发生错误，记录并通知这个错误，然后返回错误信息
		c.notifyError(err)
		return err
	}

	// 记录订单已被取消的日志
	log.Infof("[ORDER CANCELED] %s", order)

	// 函数执行成功，返回 nil 表示无错误
	return nil
}
