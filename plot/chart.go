package plot

import (
	"bytes"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/strategy"

	"github.com/StudioSol/set"
	"github.com/evanw/esbuild/pkg/api"
	log "github.com/sirupsen/logrus"
)

var (
	// staticFiles 是一个嵌入的文件系统，包含了静态资源。
	//意思就是说  embed.FS 可以把自己准备的静态样式，如css，js，等，写入到应用中
	//你正在使用 Go 1.16 或更高版本中的 embed 包来嵌入静态文件，那么确实，你需要使用 //go:embed 指令来告诉 Go 编译器哪些文件需要被嵌入到编译后的二进制文件中 go:embed assets 告诉Go编译器在编译程序时，应该将名为 assets 的文件夹中的所有内容嵌入到编译后的二进制文件中

	//go:embed assets
	staticFiles embed.FS
)

// Chart 结构体代表一个交易图表，它包含了图表的配置和数据。
type Chart struct {
	sync.Mutex                                         // 互斥锁，用于在多线程环境中保护数据一致性。
	port            int                                // 这个字段指定了图表服务监听的网络端口号。HTTP服务将使用这个端口对外提供服务。
	debug           bool                               // 一个布尔值，用于开启或关闭调试模式。在调试模式下，程序可能会显示更多的调试信息，帮助开发者了解程序运行状态或排查错误。。
	candles         map[string][]Candle                // 一个映射表（map），用来存储不同货币对的蜡烛图数据。蜡烛图是交易中常用的图表，用于表示某个时间段内的开盘价、收盘价、最高价和最低价。
	dataframe       map[string]*model.Dataframe        //一个映射表，存储每个货币对对应的Dataframe对象，键是交易对的名称，而值是与该交易对相关的数据框架（Dataframe）对象。，可以根据每个时间点给出这个交易对的数据，如最高价，最低价，交易量等等。
	ordersIDsByPair map[string]*set.LinkedHashSetINT64 //ordersIDsByPair这个结构确实就像是一个装着各种订单编号的盒子集合键（key）是货币对，比如BTC/USD或 ETH/USD，值（value）是指向 set.LinkedHashSetINT64 类型的指针，保证顺序是先后顺序，被添加的先后顺序排列，从最早添加的订单到最后添加的订单，这是一个特殊的集合，用来存储和管理订单编号。
	orderByID       map[int64]model.Order              // 一个映射表，按订单ID存储订单的详细信息。model.Order是一个结构体，包含了订单的所有相关信息，如订单类型、价格、数量等。
	indicators      []Indicator                        // 一个Indicator接口类型的切片（类似于数组但是长度可变的数据结构），用来存储图表将使用的各种指标。指标是交易分析中的工具，用来帮助交易者做出决策。
	paperWallet     *exchange.PaperWallet              //：一个指向exchange.PaperWallet类型的指针，PaperWallet模拟了一个钱包，可以用来测试交易策略而无需实际资金交易。
	scriptContent   string                             // 字符串类型，存储图表相关的JavaScript脚本内容。这些脚本在客户端执行，用于动态显示或更新图表。
	indexHTML       *template.Template                 // 一个template.Template类型的指针，它指向一个HTML模板，用于生成显示图表的网页。
	strategy        strategy.Strategy                  // 一个strategy.Strategy接口，定义了交易策略的方法，交易策略可以是任何实现了该接口的类型，用于根据市场数据生成交易信号。
	lastUpdate      time.Time                          // time.Time类型，记录了图表最后一次更新数据的时间。这可以用来判断数据是否最新，或者是否需要重新获取数据。
}

// Candle 结构体定义了蜡烛图的一个数据点。
type Candle struct {
	Time   time.Time     `json:"time"`   // 时间戳。
	Open   float64       `json:"open"`   // 开盘价。
	Close  float64       `json:"close"`  // 收盘价。
	High   float64       `json:"high"`   // 最高价。
	Low    float64       `json:"low"`    // 最低价。
	Volume float64       `json:"volume"` // 交易量。
	Orders []model.Order `json:"orders"` // 相关的订单信息。
}

// 这个Shape结构体被设计用来在图表上表示交易订单的视觉元素。它将订单的某些关键信息（如时间和价格）转换成图形化的表示，
// Shape数据和K线图有点相似都用于图表上展示交易数据，显示市场价格随时间的变化。它们共同点在于都可以视觉化地表达价格波动和市场动态。不过，它们之间有几个关键差异：1、据维度：K线图展示了开盘、收盘、最高和最低价，提供了完整的价格波动视图。Shape通常用两个点（比如起始和结束价格）来简化地显示价格变动或特定订单的范围。2、可视化形式：K线图以蜡烛形态呈现，颜色和线条区分价格方向和范围。Shape可能是线段或其他图形，展示了价格或订单信息的直观轮廓。3、用目的：K线图用于广泛分析市场趋势，适用于股票和外汇等领域。Shape则更侧重于展示具体交易信息，比如展现订单类型或执行详情，以及通过颜色区分不同情况。
// 我把数据填充到这个结构体，然后如 StartX time.Time `json:"x0"`我们定义的x0作为这数据的键，传过来的数据作为值
type Shape struct {
	StartX time.Time `json:"x0"`    // 图形的起始时间。
	EndX   time.Time `json:"x1"`    // 图形的结束时间。
	StartY float64   `json:"y0"`    // 图形的起始价位。
	EndY   float64   `json:"y1"`    // 图形的结束价位。
	Color  string    `json:"color"` // 图形的颜色。
}

// assetValue 结构体用于表示资产价值随时间的变化。
type assetValue struct {
	Time  time.Time `json:"time"`  // 时间戳。
	Value float64   `json:"value"` // 资产的价值。
}

// indicatorMetric负责提供绘制指标线的具体参数， indicatorMetric 这个代表的是线
// indicatorMetric 结构体可以用来表示各种类型的指标线，包括但不限于技术分析中常见的指标，如移动平均线（MA）、相对强弱指数（RSI）、布林带（Bollinger Bands）等。这些指标通常用于金融市场分析，帮助交易者识别市场趋势、潜在的买卖点等。
// indicatorMetric（小写开头）小写开头意味着这个结构体是私有的，仅限于定义它的包内部使用。包含JSON注解的确表明它被设计用于可能的JSON格式数据交换场景，如序列化和反序列化，常见于API响应或请求的处理中。
type indicatorMetric struct {
	Name   string      `json:"name"`  // 这个字段用来存储指标的名称，例如“移动平均线”、“相对强弱指数”等。
	Time   []time.Time `json:"time"`  // 时间戳序列。
	Values []float64   `json:"value"` // 这个字段存储每个时间点上指标的数值。与Time字段中的时间戳一一对应，这些值是绘制指标线或其他图形的基础数据。例如，如果Name字段表示“简单移动平均线”，Values可能就是这个平均线在每个时间点的计算结果。
	Color  string      `json:"color"` // 指标颜色。
	Style  string      `json:"style"` // 指标样式。
}

// plotIndicator 结构体表示一个绘制在图表上的指标。plotIndicator 结构体代表一个要在图表上绘制的完整指标。
// plotIndicator 是描述一个整体的指标的思路。它包括这个指标的名称，是否将这个指标覆盖在图表的主要部分之上（比如蜡烛图之上），以及构成这个指标的各种线（这里的“线”指的是indicatorMetric），还有开始计算这个指标之前所需准备的数据量，即预热期（Warmup）。
// 预热期:假设你想在每天市场关闭时计算过去10天的平均收盘价，以此作为一个交易信号。这个例子中，预热期是10天。这意味着，在你能开始计算第一个10日SMA之前，你需要至少10天的收盘价数据。 就是取最近十天的数据
// Name 字段在 plotIndicator 结构体中 它代表了这个指标作为一个整体的标识，例如"10日移动平均线"或"相对强弱指数 (RSI)"。这个名称用于标识或描述整个指标的作用或目的。Name 字段在 indicatorMetric 结构体中可能分别是"SMA 10日"、"SMA 20日"和"SMA 50日"，具体说明了每条线的计算周期。  就是一个是一个整体，一个代表周期性如10天..

// 如果Overlay为true：这意味着10日SMA将直接绘制在蜡烛图之上。你会看到一条线穿过蜡烛图，如果Overlay为false：这通常意味着10日SMA会在图表的另一部分单独显示，可能是在主蜡烛图的下方或上方的单独区域（称为“子图”）。这样做可以避免在主图表上过度拥挤，特别是当有多个指标同时显示时。
type plotIndicator struct {
	Name    string            `json:"name"`    // 指标名称。
	Overlay bool              `json:"overlay"` // 是否覆盖在蜡烛图之上。
	Metrics []indicatorMetric `json:"metrics"` // 指标的度量值列表。
	Warmup  int               `json:"-"`       // 预热期，这期间的指标数据可能不准确。
}

// drawdown 结构体定义了最大回撤，用于表示投资组合价值的最大下降幅度。
type drawdown struct {
	Value string    `json:"value"` // 最大回撤的百分比值。最大回撤百分比=(最低点-最高点价值)/最高点价值 x100%
	Start time.Time `json:"start"` // 标记了最大回撤开始的时间，即组合价值开始从峰值下跌的时间。
	End   time.Time `json:"end"`   // 标记了最大回撤结束的时间，即组合价值达到最低点的时间。
}

// Indicator 接口定义了所有指标必须实现的方法。
// Indicator 代表的是整个图表或者说是整个分析的框架。就像一幅画的画布，它定义了要展示的内容的范围和基本属性，比如画布的大小（图表的范围）、背景色（是否叠加在主图上）、以及需要多少准备工作（预热期）才能开始绘画（进行分析）。
type Indicator interface {
	Name() string                    // 指标的名称，这个方法返回指标的名称，如“简单移动平均线（SMA）”，“相对强弱指数（RSI）”等。
	Overlay() bool                   // 指标是否需要叠加在主图表上。意思就就是要不要单独弄一个图表，true表示在主线上，false表示不在
	Warmup() int                     // 指标计算前的预热期数据量。计算30日移动平均需要至少需要最近30天的数据。
	Metrics() []IndicatorMetric      // 指标的度量值列表。指的是各种线的列表
	Load(dataframe *model.Dataframe) // 加载数据帧以便指标计算其度量值。存放就是各种指标线的图需要的数据，充当数据的容器，它包含了计算一个或多个指标所需的全部数据，比如时间序列数据、价格信息、交易量等。
}

// IndicatorMetric 结构体代表一个特定的指标度量值。IndicatorMetric意味着它可以被包外的代码所使用。
// IndicatorMetric 则代表画布上的一根指标线，每一根都有其特定的颜色、样式和随时间变化的数据点。
type IndicatorMetric struct {
	Name   string                // 度量值的名称。
	Color  string                // 度量值在图表中的颜色。
	Style  string                // 度量值在图表中的样式。
	Values model.Series[float64] // 度量值序列，Series是一个特定的泛型类型，存储浮点数序列。
	Time   []time.Time           // 与度量值相关的时间戳序列。
}

/*
总结上面结构体：Indicator 接口是图表的规划者，定义了基本的框架和规则。 plotIndicator 结构体是根据Indicator 这个规则指定出图表 ，IndicatorMetric 结构体大写开头像是一个展馆的一个艺术品，共外人参观，这个k线可以共外部调用。indicatorMetric 结构体是小写开头像是私人作品集，k线只能是这个代码文件内部调用
*/

// ----------------------------

// OnOrder 处理一个新的订单，将其添加到图表对象的相关数据结构中。
func (c *Chart) OnOrder(order model.Order) {
	// 加锁以防止多线程或协程同时修改Chart对象，确保数据的一致性和线程安全。
	c.Lock()
	// 使用defer关键字确保在方法结束时解锁，无论是正常结束还是由于错误提前返回。
	defer c.Unlock()
	//把订单ID集中保存到一个特定的集合里，而不是直接把完整的订单对象放进去。这样做可以让我们通过订单ID在需要时快速找到、搜索和管理这些订单，这个方法不仅提高了数据处理的效率，还使得管理和维护变得更加方便和高效。
	c.ordersIDsByPair[order.Pair].Add(order.ID)
	// 把一整个订单存储到映射里面，键是订单ID，值是订单整体，这样可以通过订单ID快速找到订单实体。
	c.orderByID[order.ID] = order
}

// OnCandle 处理给定图表的进入的蜡烛数据。它锁定图表以防止并发修改，确保线程安全，然后在函数完成时解锁。
func (c *Chart) OnCandle(candle model.Candle) {
	c.Lock()
	defer c.Unlock()

	// c.candles[candle.Pair] 就是获取到当前这个交易对的所有蜡烛数据的数组，-1呢就是获得该交易对最后一个蜡烛图的数据
	lastIndex := len(c.candles[candle.Pair]) - 1

	// 如果这个蜡烛图是完整的。并且该交易对的蜡烛图没有数据，或者如果新的蜡烛数据的时间戳晚于最后一个蜡烛数据的时间戳，这意味着这个新数据确实是在最后已知数据之后发生的，因此它应该被添加到数据集中作为最新的数据点。
	// 则将其添加到对应交易对的蜡烛数据切片中。
	if candle.Complete && (len(c.candles[candle.Pair]) == 0 ||
		//如果新的蜡烛数据的时间戳晚于最后一个蜡烛数据的时间戳，这意味着这个新数据确实是在最后已知数据之后发生的，因此它应该被添加到数据集中作为最新的数据点。就是按照先后顺序接收蜡烛图数据，有利于交易者，分析市场的形式
		candle.Time.After(c.candles[candle.Pair][lastIndex].Time)) {
		//c.candles[candle.Pair] 本身就是一个切片。键是交易对，值是蜡烛图数据，如果这个蜡烛图数据是最新的，那么代码会将这个新的蜡烛数据添加到对应交易对的蜡烛数据集合中，从而更新蜡烛图数据。
		c.candles[candle.Pair] = append(c.candles[candle.Pair], Candle{
			Time:   candle.Time,
			Open:   candle.Open,
			Close:  candle.Close,
			High:   candle.High,
			Low:    candle.Low,
			Volume: candle.Volume,
			Orders: make([]model.Order, 0),
		})

		// 如果这个交易对的数据帧不存在，则初始化一个新的数据帧，并创建一个新的链接哈希集合来存储订单ID。
		if c.dataframe[candle.Pair] == nil {
			//如果这个交易对的数据帧为空，那么就创建一个数据帧实例。
			c.dataframe[candle.Pair] = &model.Dataframe{
				//air字段被设置为当前处理的交易对名称（candle.Pair
				Pair: candle.Pair,
				//Metadata字段通过构建一个映射（map）结构，创建了一个灵活的数据存储空间。这个映射的键是字符串，用于唯一标识和命名不同的数据集，例如“平均成交价”或“日交易量”。与每个键相关联的值是model.Series[float64]类型的泛型切片，这意味着对于每个标识符，Metadata都能够存储一个浮点数序列，这些序列代表了随时间变化的数据点集合。
				//它可以容纳交易对的各种附加信息，使得用户能够跟踪和分析特定指标随时间的演变。，例如果你对某个交易对的移动平均线感兴趣，可以在Metadata中为这个指标创建一个序列，并将计算出的每日移动平均值存储为序列中的数据点。
				Metadata: make(map[string]model.Series[float64]),
			}
			//这行代码为特定的交易对初始化一个新的有序集合，用来存储该交易对的订单ID。 键是交易对，值是set.NewLinkedHashSetINT64()类型可以存储交易的订单编号这个有序集合保持了订单ID的插入顺序，set.NewLinkedHashSetINT64()类型 ，保证订单在集合中是按照它们被添加的顺序排列的，从最先添加的订单到最后添加的订单
			c.ordersIDsByPair[candle.Pair] = set.NewLinkedHashSetINT64()
		}

		// 更新数据帧中的相关字段，包括收盘价、开盘价、最高价、最低价、成交量和时间。
		//就是把这些蜡烛图的数据，放到数据帧里面，然后呢根据时间看历史数据，查看历史，分析趋势
		c.dataframe[candle.Pair].Close = append(c.dataframe[candle.Pair].Close, candle.Close)
		c.dataframe[candle.Pair].Open = append(c.dataframe[candle.Pair].Open, candle.Open)
		c.dataframe[candle.Pair].High = append(c.dataframe[candle.Pair].High, candle.High)
		c.dataframe[candle.Pair].Low = append(c.dataframe[candle.Pair].Low, candle.Low)
		c.dataframe[candle.Pair].Volume = append(c.dataframe[candle.Pair].Volume, candle.Volume)
		c.dataframe[candle.Pair].Time = append(c.dataframe[candle.Pair].Time, candle.Time)
		c.dataframe[candle.Pair].LastUpdate = candle.Time

		// 对于蜡烛数据中的每个元数据键值对，更新数据帧中的元数据。
		//k 代表的是键可以是"MA10"（表示10天移动平均线）或"Volume"（表示成交量）等，用来指代存储在Metadata中的不同类型的数据。v 代表的是值（Value）如某一时刻的10天移动平均线的具体数值或那一时刻的成交量等。
		//将值v存储到指定交易对的数据帧（dataframe）中，具体地，存储到该数据帧的Metadata字段下的k键对应的序列中。
		for k, v := range candle.Metadata {
			c.dataframe[candle.Pair].Metadata[k] = append(c.dataframe[candle.Pair].Metadata[k], v)
		}
		// 更新Chart 结构体(代表一个交易图表，它包含了图表的配置和数据)最后更新的时间 。这对于维护数据的实时性非常重要，尤其是在需要频繁更新数据以反映最新市场动态的交易系统中。
		c.lastUpdate = time.Now()
	}
}

// equityValuesByPair方法用于从交易表Chart结构体中提取资产价值和权益价值的时间序列，时间序列的意思是记录在不同时间交易表Chart结构体中，资产价值和，权益价值的数据
// 参数pair是一个字符串，表示要查询的交易对，比如"BTC/USD"。
// 方法返回两个切片：assetValues和equityValues，分别代表资产价值和权益价值的时间序列。
func (c *Chart) equityValuesByPair(pair string) (assetValues []assetValue, equityValues []assetValue) {
	// 初始化两个空切片，用于存储资产价值和权益价值的时间序列数据。
	assetValues = make([]assetValue, 0)
	equityValues = make([]assetValue, 0)

	// 检查Chart结构体中的paperWallet字段是否存在。先检查是否存在虚拟钱包
	if c.paperWallet != nil {
		// 如果存在，使用exchange.SplitAssetQuote函数切割交易对，获取资产部分。
		asset, _ := exchange.SplitAssetQuote(pair)
		// 遍历指定虚拟钱包交易对资产的历史交易记录，再把遍历到的基础资产的时间戳，资产价值，放到存放基础资产的切片中
		for _, value := range c.paperWallet.AssetValues(asset) {
			assetValues = append(assetValues, assetValue{
				Time:  value.Time,
				Value: value.Value,
			})
		}

		// 遍历虚拟钱包的总余额，把遍历到的时间戳，报价资产放到报价资产切片中
		for _, value := range c.paperWallet.EquityValues() {
			equityValues = append(equityValues, assetValue{
				Time:  value.Time,
				Value: value.Value,
			})
		}
	}

	// 返回资产价值和权益价值(交易账户的总价值)的时间序列。
	return assetValues, equityValues
}

// indicatorsByPair 方法用于提取指定交易对的所有技术指标。
// 参数pair是一个字符串，指定了要查询的交易对，例如"BTC/USD"。
func (c *Chart) indicatorsByPair(pair string) []plotIndicator {
	// 初始化一个空的plotIndicator切片，用于存储所有指标的配置和数据。
	indicators := make([]plotIndicator, 0)

	// 遍历Chart结构体中预先配置的指标列表。指标线列表有着指标线的定义规则。
	for _, i := range c.indicators {
		// 贮存了指定交易对相关的时间序列数据，包括最高价、最低价、开盘价、收盘价、成交量等信息。这些数据随时间排列，形成了一个数据帧（DataFrame）。
		// Load函数将每个时间段的数据帧数据应用到特定的指标计算中，然后在图表上展示这些指标，帮助交易者更好地观察和理解市场的动态和趋势。
		i.Load(c.dataframe[pair])

		// 构造一个新的plotIndicator对象，存储当前指标的名称、是否为图表覆盖层、需要的预热期和指标的具体度量值。
		// 将i(Indicator)里面定义的规则 应用到plotIndicator 对象
		indicator := plotIndicator{
			Name:    i.Name(),                   // 指标名称
			Overlay: i.Overlay(),                // 指示该指标是否应该被覆盖在主图表上
			Warmup:  i.Warmup(),                 // 指标计算开始前需要的数据点数量（预热期）
			Metrics: make([]indicatorMetric, 0), // 初始化指标的度量值数组
		}

		// 遍历当前指标的所有度量值(各种线)，并添加到indicator对象中Metrics。
		for _, metric := range i.Metrics() {
			indicator.Metrics = append(indicator.Metrics, indicatorMetric{
				Name:   metric.Name,   // 度量值的名称
				Values: metric.Values, // 度量值的具体数值数组
				Time:   metric.Time,   // 度量值对应的时间点
				Color:  metric.Color,  // 度量值绘图时使用的颜色
				Style:  metric.Style,  // 度量值绘图时使用的样式
			})
		}

		// 将构造好的indicator对象添加到indicators切片中。
		indicators = append(indicators, indicator)
	}

	// 如果策略存在，则从策略中提取额外的指标配置和数据。
	if c.strategy != nil {
		warmup := c.strategy.WarmupPeriod()                            // 策略开始前需要最近如几天天、几个小时的预热期数据 。
		strategyIndicators := c.strategy.Indicators(c.dataframe[pair]) // 获取策略在指定交易对上计算后得到的指标列表。例如，可能会得到一组移动平均线交叉指标，其中包含每次交叉发生的时间、价格、交叉类型等信息。

		// 遍历策略指标，并进行类似的处理。
		for _, i := range strategyIndicators {
			indicator := plotIndicator{
				Name:    i.GroupName,                // 策略指标组的名称
				Overlay: i.Overlay,                  // 指示该指标是否应该被覆盖在主图表上
				Warmup:  i.Warmup,                   // 指标计算开始前需要的数据点数量（预热期）
				Metrics: make([]indicatorMetric, 0), // 初始化指标的度量值数组
			}

			// 只包括预热期之后的度量值。
			for _, metric := range i.Metrics {
				//如果当前度量值的数据点数量小于策略预热期所需的数据点数量，则跳过该度量值，因为数据量不足无法进行有效分析。
				if len(metric.Values) < warmup {
					continue // 如果度量值的数量少于预热期，跳过该度量值。
				}
				//如我去最近五天的预热值作为参考，在展示指标时，常常会去除预热值之前的数据点，以便更好的呈现真实数据。
				indicator.Metrics = append(indicator.Metrics, indicatorMetric{
					Time:   i.Time[i.Warmup:],        // 去除预热期之前的时间点
					Values: metric.Values[i.Warmup:], // 去除预热期之前的度量值
					Name:   metric.Name,              // 度量值的名称
					Color:  metric.Color,             // 度量值绘图时使用的颜色
					Style:  string(metric.Style),     // 度量值绘图时使用的样式
				})
			}
			indicators = append(indicators, indicator) // 将构造好的indicator对象添加到indicators切片中。
		}
	}
	// 返回包含了所有指标配置和数据的切片，供后续绘图或分析使用。
	return indicators
}

// candlesByPair 根据货币对返回相关的蜡烛图数据（Candle）切片。
func (c *Chart) candlesByPair(pair string) []Candle {
	// 初始化与指定货币对的蜡烛图数据等长的Candle切片。c.candles映射中对应pair键（即货币对字符串）的蜡烛图数据切片的长度，确保新创建的candles切片长度与之相等。
	//长度（len 函数返回的值）指的是该货币对的蜡烛图数据中蜡烛图的数量。每个蜡烛图代表一个特定的时间段（如1分钟、1小时、1天等），所以这个长度实际上表示了有多少个这样的时间段被记录下来了。这个数字是动态变化的，因为随着时间的推移，会有更多的蜡烛图数据被添加进来。
	candles := make([]Candle, len(c.candles[pair]))
	// 初始化orderCheck这个map，使用map追踪每个订单是否已被处理。键（key）是订单的ID（类型为int64），而其值（value）是一个布尔值（bool），表示该订单是否已经被处理过
	orderCheck := make(map[int64]bool)
	// 标记所有与该货币对相关的订单ID为未处理。
	//.Iter()方法和直接遍历的效果一样，使用.Iter()方法的原因可能是c.ordersIDsByPair[pair]不是一个直接可遍历的类型，或者设计者想通过.Iter()提供一种特定的遍历方式（比如确保遍历顺序或者在遍历过程中提供其他额外的功能）。
	//因为ordersIDsByPair的类型为map[string]*set.LinkedHashSetINT64 保持了元素的唯一性，又保留了插入顺序。这意味着，与简单的切片、数组或普通映射不同 ，所以不直接遍历，用.Iter()方法遍历依次访问id的好处是，保持顺序：通过迭代器遍历可以确保处理订单ID的顺序与它们被添加到集合中的顺序一致，同时保留了代码的清晰度和扩展性。

	for id := range c.ordersIDsByPair[pair].Iter() {
		//遍历出来的id 变成true表示已经处理过了
		orderCheck[id] = true
	}

	// 遍历指定货币对的每个蜡烛图数据。这段代码的目的是将与特定货币对相关的订单根据它们的更新时间归类到正确的蜡烛图数据中。每个蜡烛图代表了一个时间段内的市场活动，包括价格的开盘、收盘、最高和最低点等。通过将订单归类到相应的蜡烛图中，可以更准确地分析在该时间段内的交易活动。
	for i := range c.candles[pair] {
		// 这行代码将原始的蜡烛图数据复制到一个新的切片中。这是为了创建一个处理后的蜡烛图数据副本，避免直接修改原始数据。
		candles[i] = c.candles[pair][i]
		// c.ordersIDsByPair[pair].Iter() 因为这个订单集合里面本来就是有顺序的所以通过 .Iter() 方法遍历与该货币对相关的所有订单ID。
		for id := range c.ordersIDsByPair[pair].Iter() {
			//获取订单的详情
			order := c.orderByID[id]
			//如果订单的更新时间晚于当前蜡烛图(i)开始时间，又早于下一个拉组图(i+1)的开始时间，那么得出这个订单属于蜡烛图(i)，如果订单恰好等于当前蜡烛图(i)的开始时间，这个订单属于蜡烛图(i)
			//如订单的更新时间是10:00 , 蜡烛图有三个数据，分别是9:00-10:00 , 10:00-11:00 , 11:00-12:00, 因为判断订单的更新时间比当前蜡烛图的开始时间晚，那只能是10:00过后，下一个条件，订单的更新时间比下一个蜡烛图的开始时间早，所以例子中只能是11:00之前，因为传过来的订单更新时间会带入判断，所以得出的结论是，在10:00-11:00之间
			if i < len(c.candles[pair])-1 &&
				(order.UpdatedAt.After(c.candles[pair][i].Time) &&
					order.UpdatedAt.Before(c.candles[pair][i+1].Time)) ||
				//或者是当订单正好等于当前的蜡烛图的开始时间如10：00, 那么蜡烛图的更新时间在当前蜡烛图内
				order.UpdatedAt.Equal(c.candles[pair][i].Time) {
				//从 orderCheck 映射中移除该订单ID，标记该订单已经被处理。orderCheck 映射是用来跟踪哪些订单已经被处理过的，以防止同一个订单被重复添加到多个蜡烛图中。
				delete(orderCheck, id)
				//将符合条件的订单添加到当前蜡烛图的订单列表中。这表示该订单发生在当前蜡烛图代表的时间范围内，应该被包含在这个蜡烛图的数据里。
				candles[i].Orders = append(candles[i].Orders, order)
			}
		}
	}

	// 对于仍标记为未处理的订单，如果订单更新时间在最后一个蜡烛图之后，
	// 则添加到最后一个蜡烛图的订单列表中。
	for id := range orderCheck {
		order := c.orderByID[id]
		if order.UpdatedAt.After(c.candles[pair][len(c.candles)-1].Time) {
			c.candles[pair][len(c.candles)-1].Orders = append(c.candles[pair][len(c.candles)-1].Orders, order)
		}
	}

	// 返回处理后的蜡烛图数据切片。
	return candles
}

// shapesByPair 为指定的货币对生成图形数据。这个图形界面上可视化订单数据X轴表示的是时间轴，它用来显示订单的创建时间和更新时间，Y轴则用来表示订单的价格信息。图形的起始点（StartY）表示订单的参考价格，而结束点（EndY）表示订单的实际价格。 过将这些信息绘制在图形界面上，用户可以快速、直观地理解订单随时间的变化情况，包括订单价格的波动以及订单的持续期。这种可视化方法尤其有助于分析和决策，比如评估哪些时间段更活跃、价格波动较大等，进而对交易策略进行调整。
func (c *Chart) shapesByPair(pair string) []Shape {
	// 初始化一个空的Shape切片，用于存储最终的图形数据。
	shapes := make([]Shape, 0)

	// 遍历该货币对的所有订单ID。
	for id := range c.ordersIDsByPair[pair].Iter() {
		// 根据ID获取订单的详细信息。
		order := c.orderByID[id]

		// 如果订单类型不是止损单（StopLoss）和限价挂单（LimitMaker），则跳过当前循环。
		// 意思就是我只关注限价挂单，还有止损单，这两个，关注太多，结果不够清晰不够高效
		if order.Type != model.OrderTypeStopLoss &&
			order.Type != model.OrderTypeLimitMaker {
			continue
		}

		// 创建一个新的Shape，用订单的创建时间和更新时间作为X轴的开始和结束点，
		// 使用参考价格和订单价格作为Y轴的开始和结束点，初始颜色设置为绿色。
		//把符合条件的订单放到shape图中观察
		shape := Shape{
			StartX: order.CreatedAt,
			EndX:   order.UpdatedAt,
			StartY: order.RefPrice,
			EndY:   order.Price,
			Color:  "rgba(0, 255, 0, 0.3)",
		}

		// 如果订单类型是止损（StopLoss），则将颜色改为红色。
		if order.Type == model.OrderTypeStopLoss {
			shape.Color = "rgba(255, 0, 0, 0.3)"
		}

		// 将创建的Shape添加到shapes切片中。
		shapes = append(shapes, shape)
	}

	// 返回包含所有创建的Shape的切片。
	return shapes
}

// orderStringByPair 根据指定的货币对，生成一个包含订单信息的二维字符串切片。
// 这个方法实质上是在创建一个类似于二维表格的数据结构，其中包含了特定货币对相关的订单信息。每一行代表一个订单，每一列则对应该订单的一个具体属性，比如订单的创建时间、状态、买卖方向、订单ID、订单类型、数量、价格、总价和利润等。这种组织数据的方式使得信息易于存取、显示和分析，提供了一个清晰的视图来查看和处理订单数据。
// 如创建时间	状态	买/卖	订单ID	订单类型	数量	价格	总价	利润
// 2024-03-28	已完成	买入	1	限价订单	100	10.00	1000.00	50.00
// 2024-03-27	待定	卖出	2	市价订单	200	9.50	1900.00	-20.00
func (c *Chart) orderStringByPair(pair string) [][]string {
	// 初始化一个空的二维字符串切片，用来存储订单信息。
	//切片里面又套着另一个切片，就是二维字符串切片
	/*
		如：orders := [][]string{
		    {"2024-03-28", "已完成", "买入", "1", "限价订单", "100", "10.00", "1000.00", "50.00"},
		    {"2024-03-27", "未完成", "卖出", "2", "市价订单", "200", "9.50", "1900.00", "-20.00"},
		}
	*/
	orders := make([][]string, 0)

	// 遍历该货币对相关的所有订单ID。
	for id := range c.ordersIDsByPair[pair].Iter() {
		// 根据订单ID获取订单详细信息。
		o := c.orderByID[id]

		// 初始化一个空字符串用于表示订单利润。
		var profit string
		// 如果订单利润不为0，则将其格式化为字符串，保留两位小数。
		if o.Profit != 0 {
			profit = fmt.Sprintf("%.2f", o.Profit)
		}

		// 将订单的各个属性格式化为一个长字符串，属性之间用逗号分隔。
		// 包括创建时间、状态、买卖方向、订单ID、订单类型、数量、价格、总价和利润。
		orderString := fmt.Sprintf("%s,%s,%s,%d,%s,%f,%f,%.2f,%s",
			o.CreatedAt, o.Status, o.Side, o.ID, o.Type, o.Quantity, o.Price, o.Quantity*o.Price, profit)

		// 将上述长字符串以逗号为分隔符拆分成子字符串数组。
		//如:[]string{"2024-03-28", "已完成", "买入", "1", "限价订单", "100", "10.00", "1000.00", "50.00"}

		order := strings.Split(orderString, ",")

		// 将这个订单的字符串数组添加到二维字符串切片中。
		orders = append(orders, order)
	}

	// 返回包含所有订单信息的二维字符串数组。
	return orders
}

// handleHealth 是 Chart 类型的方法，用于响应健康检查的HTTP请求。
// 我们自己制作的机器人利用Chart制作交易图表，但是利用太多Chart制作图表的话会负载过重，保证Chart制作成功，所以每个Chart实现handleHealth方法，用来响应来自负载均衡器的健康检查请求。不健康状态：如果无法正常获取市场数据，handleHealth方法会首先尝试向响应中写入上次更新的时间，然后设置HTTP状态码为503 Service Unavailable，通知负载均衡器这个实例当前无法处理请求。健康状态：如果自上次更新以来的时间没有超过1小时10分钟，方法认为服务正常工作，并设置HTTP状态码为200 OK
// w http.ResponseWriter：这是一个用于写入HTTP响应的接口。通过这个接口，我们可以将数据写入HTTP响应体，发送给HTTP客户端。
// _ *http.Request：这是一个表示HTTP请求的结构体指针。在这个方法中，我们没有使用到这个参数，所以使用下划线_来表示我们不需要它。
func (c *Chart) handleHealth(w http.ResponseWriter, _ *http.Request) {
	// 检查自最后更新以来是否已经过去了超过1小时10分钟。
	//time.Since计算间隔时间这里c.lastUpdate自最后一次更新到现在的间隔
	if time.Since(c.lastUpdate) > time.Hour+10*time.Minute {
		//如果服务不健康，则尝试向响应写入上次更新的时间。这里使用c.lastUpdate.String()来获取上次更新时间的字符串表示，并将其转换为字节切片写入了HTTP响应的正文中。
		_, err := w.Write([]byte(c.lastUpdate.String()))
		// 检查写入操作是否出错，如果有错误，使用log.Error记录错误。
		if err != nil {
			log.Error(err)
		}
		// 设置HTTP响应状态码为503 Service Unavailable，
		// 表示服务目前无法处理请求。
		w.WriteHeader(http.StatusServiceUnavailable)
		// 由于已经确定服务不健康，提前返回，不再执行后续代码。
		return
	}
	// 如果自上次更新以来的时间未超过1小时10分钟，
	// 表示服务处于健康状态，设置HTTP响应状态码为200 OK。
	w.WriteHeader(http.StatusOK)
}

// handleIndex 是 Chart 类型的方法，用于处理首页的 HTTP 请求。
// 这段代码处理了两种情况：一种是用户未指定货币对，自动重定向到一个默认的货币对页面；另一种是用户指定了货币对，或者在没有默认货币对可用时访问首页，这时会渲染并显示一个包含货币对信息的页面。
func (c *Chart) handleIndex(w http.ResponseWriter, r *http.Request) {
	// 初始化一个空切片用于存储所有货币对的名称。
	var pairs = make([]string, 0, len(c.candles))
	// 遍历所有货币对，将其名称添加到 pairs 切片中。
	for pair := range c.candles {
		pairs = append(pairs, pair)
	}

	// 对 pairs 切片进行排序，我按这个排序切片里面的货币会按照AZ排序
	sort.Strings(pairs)

	// 从 HTTP 请求的查询参数中获取货币对。
	//这种方式通常用于处理GET请求中的查询参数，或者处理某些POST请求中URL的查询参数。在这个场景下，它被用来获取用户通过URL指定的货币对名称。例如，如果用户访问的URL是 http://example.com/?pair=EURUSD，那么 r.URL.Query().Get("pair") 将返回 "EURUSD"，即用户想要查询或操作的货币对。
	pair := r.URL.Query().Get("pair")
	// 如果查询参数中没有指定货币对，并且 pairs 切片不为空，则重定向到第一个货币对的首页。。如果你在开发一个货币对交易网站，len(pairs) > 0 这个条件表示你的网站已经有一些货币对可供查询。
	if pair == "" && len(pairs) > 0 {
		//服务器端：使用 w 发送一个重定向响应，告诉客户端去访问一个新的URL。客户端（如浏览器）：接收到重定向响应后，自动根据响应中的 Location 头部指定的新URL发起一个新的请求。用户看到的效果就是浏览器地址栏中的URL发生了变化，并加载了新URL对应的页面内容。
		//假设 pairs[0] 的值为 "USD/EUR"，那么 fmt.Sprintf("/?pair=%s", pairs[0]) 的结果将是 "/?pair=USD/EUR"。这个结果是一个构造好的URL，它将在重定向中被用作目的地地址。
		//例如：http://example.com/?pair=EURUSD，
		//http.StatusFound 状态码302表示产生了一个重定向
		http.Redirect(w, r, fmt.Sprintf("/?pair=%s", pairs[0]), http.StatusFound)
		return
	}

	//设置响应头 Content-Type 为 text/html 的确是在告诉浏览器或接收数据的客户端，响应的内容类型（即文本格式）是HTML。
	w.Header().Add("Content-Type", "text/html")
	// c.indexHTML(*template.Template)模板的Execute方法，根据提供的数据将一个包含键值对的map（pair和pairs）， 动态生成HTML页面内容，并通过HTTP响应发送给客户端。
	err := c.indexHTML.Execute(w, map[string]interface{}{
		"pair":  pair,
		"pairs": pairs,
	})
	// 检查渲染过程是否出错，如果有错误，使用 log.Error 记录错误信息。
	if err != nil {
		log.Error(err)
	}
}

// handleData方法的作用是从后端系统中提取用户请求的货币对数据，处理并以标准化的格式（JSON）返回给前端，使用户能够通过图形化界面获取到丰富、实时的市场数据。这种前后端分离的架构增强了应用的灵活性和扩展性，同时提升了用户体验。
func (c *Chart) handleData(w http.ResponseWriter, r *http.Request) {
	// 从URL查询参数中获取`pair`，它表示用户请求的货币对。
	pair := r.URL.Query().Get("pair")

	// 如果没有指定货币对，则返回404 Not Found状态码并结束处理。
	//w.WriteHeader 是Go语言中用于设置HTTP响应的状态码的方法，http.StatusNotFound用设置HTTP响应的状态码为404，并且立即结束处理这个请求，不再执行后续的代码。
	if pair == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	//Add() 方法会添加一个新的字段，而 Set() 方法会覆盖已有字段的值。
	// 设置HTTP响应的Content-Type头字段为"text/json"，表示返回给客户端的内容是以JSON格式编码的数据。
	w.Header().Set("Content-type", "text/json")

	// 初始化一个指向`drawdown`结构体的指针变量，用于存储最大回撤信息。
	var maxDrawdown *drawdown
	// c.paperWallet 表示虚拟钱包对象，如果不为空，就意味着有相关的数据可以用来计算最大回撤。
	if c.paperWallet != nil {
		value, start, end := c.paperWallet.MaxDrawdown()
		maxDrawdown = &drawdown{
			Start: start,                          // 最大回撤开始时间
			End:   end,                            // 最大回撤结束时间
			Value: fmt.Sprintf("%.1f", value*100), // 这段代码的目的就是将拿到的最大回撤百分比值转换为小数形式，并保留一位小数，然后将其格式化为字符串。
		}
	}

	// 分割货币对为资产和报价货币。
	asset, quote := exchange.SplitAssetQuote(pair)
	// 获取资产值和权益值的时间序列数据。
	assetValues, equityValues := c.equityValuesByPair(pair)
	// 编码并发送JSON响应给客户端，包含了货币对的各种数据信息。
	//json.NewEncoder(w)：这一部分创建了一个新的JSON编码器（Encoder), W通常是HTTP响应对象 http.ResponseWriter，这个方法会将提供的数据编码为JSON格式，并写入到 w 中。Encode(...)：这部分调用了编码器的 Encode 方法，传入了一个 map[string]interface{} 类型的参数。这个方法会将而是将数据编码为JSON格式，并将其写入到 w 中。
	err := json.NewEncoder(w).Encode(map[string]interface{}{
		"candles":       c.candlesByPair(pair),    // 货币对的蜡烛图数据
		"indicators":    c.indicatorsByPair(pair), // 货币对的技术指标数据
		"shapes":        c.shapesByPair(pair),     // 货币对的形状数据（可能用于标记图表）
		"asset_values":  assetValues,              // 资产值时间序列数据
		"equity_values": equityValues,             // 权益值时间序列数据(交易账户的总价值)
		"quote":         quote,                    // 报价货币
		"asset":         asset,                    // 资产货币
		"max_drawdown":  maxDrawdown,              // 最大回撤信息
	})
	// 如果在编码JSON时遇到错误，则记录错误信息。
	if err != nil {
		log.Error(err)
	}
}

// handleTradingHistoryData 是 Chart 类型的方法，用于处理交易历史数据的 HTTP 请求。
func (c *Chart) handleTradingHistoryData(w http.ResponseWriter, r *http.Request) {
	// 从 HTTP 请求的查询参数中获取货币对。
	pair := r.URL.Query().Get("pair")
	// 如果没有提供货币对，则返回404 Not Found状态码，并结束处理。
	if pair == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 设置响应头的 Content-type 为 text/csv，表示返回的数据是 CSV 格式。
	w.Header().Set("Content-type", "text/csv")
	// 意思就是服务器设置HTTP 响应头的 Content-Disposition 字段为 "attachment;filenam，用户不需要执行任何操作，只要点击了下载链接或触发了下载动作，浏览器就会自动开始下载文件。通常情况下，浏览器会将文件保存到默认的下载目录中 然后保存到本地的文件名是history_{pair}.csv  {pair} 动态货币名称
	w.Header().Set("Content-Disposition", "attachment;filename=history_"+pair+".csv")
	// 设置响应头的 Transfer-Encoding 为 chunked，表示采用分块传输编码方式传输数据。
	//实际上，分块传输编码是一种在 HTTP 传输过程中使用的一种传输编码方式，它允许服务器在发送响应时将数据分成多个小块（即数据块），每个数据块都会带有长度信息。当浏览器接收到这些数据块时，它会逐个处理这些数据块，并在接收到每个数据块时立即开始处理，而不必等待整个文件下载完成。这样可以有效地减少等待时间，让用户可以更快地开始访问文件内容。
	w.Header().Set("Transfer-Encoding", "chunked")

	// 获取特定货币对的订单数据，并准备将其转换为 CSV 格式。
	orders := c.orderStringByPair(pair)

	// 创建一个新的字节缓冲区，用于暂存将要生成的 CSV 数据。
	buffer := bytes.NewBuffer(nil)
	// 创建一个 CSV 编写器，用于按照 CSV 格式处理数据并将其写入到字节缓冲区中。此时，缓冲区是空的，等待数据被写入。
	csvWriter := csv.NewWriter(buffer)
	// 写入 CSV 文件的头部信息。
	// 这行代码调用了csvWriter的Write方法，传入一个字符串切片作为参数。这个切片包含了CSV文件的头部信息，也就是每一列的标题。这意味着在CSV文件中，第一行将会是这些标题，分别是created_at, status, side, id, type, quantity, price, total, profit。这些标题代表了接下来CSV文件中每行数据的字段名称，例如交易创建时间、状态、买卖方向、交易ID、交易类型、数量、价格、总计、利润等。
	err := csvWriter.Write([]string{"created_at", "status", "side", "id", "type", "quantity", "price", "total", "profit"})
	if err != nil {
		// 如果写入头部信息出错，则记录错误并返回 400 Bad Request 状态码。
		log.Errorf("failed writing header file: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 将订单数据写入到 CSV 文件中。
	//当你调用csvWriter.WriteAll(orders)时，你告诉csvWriter把orders（一个包含订单数据的切片，其中每个元素都是一个代表单条记录的字符串数组）以CSV格式写入。
	// WriteAll方法会遍历orders中的每一项，将它们转换成CSV格式，并写入到csvWriter关联的buffer中。
	err = csvWriter.WriteAll(orders)
	if err != nil {
		// 如果写入订单数据出错，则记录错误并返回 400 Bad Request 状态码。
		log.Errorf("failed writing data: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// 刷新 CSV 编写器，确保所有数据都写入到缓冲区中。
	//调用csvWriter.Flush()的作用是确保所有缓存在csvWriter内部的数据都被推送到它关联的字节缓冲区buffer中。在写入CSV数据时，出于性能考虑，数据可能会被先缓存起来，而不是直接写入到输出流（在这个场合，就是buffer）。Flush方法就是用来清空这个内部缓存，强制所有待写数据立即写入到指定的输出流。
	csvWriter.Flush()

	// 设置响应头的状态码为 200 OK。
	w.WriteHeader(http.StatusOK)
	// 将缓冲区中的 CSV 数据写入到 HTTP 响应中。
	//buffer.Bytes() 相当把之前存的数据，变成一个复制一个副本拿出来，然后再写入响应中传给前端
	_, err = w.Write(buffer.Bytes())
	if err != nil {
		// 如果写入响应数据出错，则记录错误并返回 400 Bad Request 状态码。
		log.Errorf("failed writing response: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// 这个Start方法的作用是启动一个Web服务器，并设置了特定的路由和处理函数来响应不同的HTTP请求。
func (c *Chart) Start() error {
	// 就是设置一个staticFiles文件引用assets静态文件，表示staticFiles等于assets文件，而这行代码告诉Web服务器，任何以/assets/开头的HTTP请求都应该通过查找staticFiles来响应，staticFiles就相当于assets文件
	// 不直接去assets 找文件，要设置一个引用staticFiles，好处抽象层级：通过设置引用，你为资源提供了一个抽象层，这意味着你的Web应用代码不直接依赖于文件系统的具体布局。安全隔离：直接在文件系统上操作时，需要谨慎确保不会暴露敏感信息。统一管理：通过统一的接口管理静态资源，可以方便地实现如缓存策略、权限检查等高级功能。

	//http.FS就是一个适配器，它允许http.FileServer能够访问使用fs.FS接口的文件系统。staticFiles在这里是一个变量，它实现了fs.FS接口。这意味着staticFiles代表的可以是实际的文件系统目录当你将它们组合在一起使用时，http.FileServer(http.FS(staticFiles))创建了一个处理器，这个处理器能够接收HTTP请求，并从staticFiles指向的文件系统中查找和返回请求的文件。
	http.Handle(
		"/assets/",
		http.FileServer(http.FS(staticFiles)),
	)

	//当Web服务器收到一个请求，这个请求的URL路径是/assets/chart.js时通过 通过匿名函数设置响应头，"application/javascript" 告诉前端，返回的是一个js文件接下来，处理函数使用fmt.Fprint(w, c.scriptContent)将c.scriptContent变量中的内容写入响应体中。这里的c.scriptContent是一个字符串变量，包含了要发送给客户端的JavaScript代码。这段代码可能是静态定义的，也可能是动态生成的，取决于你的应用逻辑。
	// 如客服端 <script src="http://localhost:8080/assets/chart.js"></script>
	http.HandleFunc("/assets/chart.js", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-type", "application/javascript") // 设置响应的内容类型为JavaScript。
		fmt.Fprint(w, c.scriptContent)                           // 将c.scriptContent写入响应中，发送给客户端。
	})

	// 为"/health"路径设置处理函数。用于检查服务器健康状态。
	http.HandleFunc("/health", c.handleHealth)

	// 为"/history"路径设置处理函数。用于处理与交易历史数据相关的请求。
	http.HandleFunc("/history", c.handleTradingHistoryData)

	// 为"/data"路径设置处理函数。可以用于返回实时数据或其他信息。
	http.HandleFunc("/data", c.handleData)

	// 设置根路径"/"的处理函数。这通常用于返回应用程序的主页面或仪表板。
	http.HandleFunc("/", c.handleIndex)

	// 打印一条消息到控制台，告知服务在哪个端口上可用。
	fmt.Printf("Chart available at http://localhost:%d\n", c.port)

	// 启动HTTP服务器监听在指定端口上。`ListenAndServe`会阻塞，直到服务器停止，如果启动成功，它将永远不会返回非nil的错误。
	//这个函数需要两个参数：第一个是字符串类型的地址，它指定服务器监听的网络地址和端口号；第二个参数是http.Handler，它是一个接口，用于处理所有的HTTP请求。如果这个参数是nil，那么服务器将使用默认的多路复用器http.DefaultServeMux作为其处理器。
	//如果c.port的值是8080，那么fmt.Sprintf(":%d", c.port)的结果就是字符串":8080"

	//这段代码return http.ListenAndServe(fmt.Sprintf(":%d", c.port), nil)的确是在启动一个监听在c.port指定端口号的HTTP服务，并且它使用默认的多路复用器http.DefaultServeMux作为其请求处理器。
	return http.ListenAndServe(fmt.Sprintf(":%d", c.port), nil)
}

// Option 类型定义了一个函数签名，用于配置Chart实例的选项。
// 这是函数式选项模式的核心，允许以灵活的方式设置或修改Chart的各种属性。
// type Option func(*Chart)是定义了一种可以被用来修改Chart实例的函数的类型 名字是Option  所以调用Option  就是调用这个函数
type Option func(*Chart)

// WithPort 创建一个配置选项，用于设置Chart实例监听的端口号。
// 这对于Web服务器或需要网络监听的应用来说是必要的配置。
func WithPort(port int) Option {
	return func(chart *Chart) {
		chart.port = port
	}
}

// WithStrategyIndicators 创建一个配置选项，用于为Chart设置交易策略指标。
// 这可以根据不同的交易策略来动态调整Chart的行为或表现。
func WithStrategyIndicators(strategy strategy.Strategy) Option {
	return func(chart *Chart) {
		chart.strategy = strategy
	}
}

// WithPaperWallet 创建一个配置选项，用于为Chart设置纸钱包。
// 纸钱包通常用于加密货币交易中，存储资金的一种安全方式。
func WithPaperWallet(paperWallet *exchange.PaperWallet) Option {
	return func(chart *Chart) {
		chart.paperWallet = paperWallet
	}
}

// WithDebug 创建一个配置选项，开启Chart的调试模式。
// 在调试模式下，Chart可能会提供更详细的日志输出，或者禁用某些性能优化，以便于开发和调试。
func WithDebug() Option {
	return func(chart *Chart) {
		chart.debug = true
	}
}

// 这段代码允许你为 Chart 实例自定义指标的规则。通过使用 WithCustomIndicators 函数，你可以传入一个或多个 Indicator 类型的参数，
// 这允许开发者根据特定需求定制Chart的分析指标，提高了Chart的灵活性和适用性。
func WithCustomIndicators(indicators ...Indicator) Option {
	return func(chart *Chart) {
		chart.indicators = indicators
	}
}

// NewChart 创建一个新的Chart实例，并根据提供的选项进行配置。
// 这个函数接收一系列Option函数作为参数，这些函数用于定制Chart实例的配置。
// options ...Option这种参数定义方式允许NewChart函数接受任意数量的Option类型的函数作为参数。这些Option函数代表了不同的配置选项，每个都能以特定的方式修改Chart实例的状态或属性。
func NewChart(options ...Option) (*Chart, error) {
	// 初始化Chart实例，设置默认值。
	// 包括默认端口8080，以及初始化几个重要的数据结构，如candles, dataframe等。
	chart := &Chart{
		port:            8080,                                     // 默认端口8080
		candles:         make(map[string][]Candle),                // 初始化candles映射
		dataframe:       make(map[string]*model.Dataframe),        // 初始化dataframe映射
		ordersIDsByPair: make(map[string]*set.LinkedHashSetINT64), // 初始化ordersIDsByPair映射
		orderByID:       make(map[int64]model.Order),              // 初始化orderByID映射
	}

	// 遍历options切片，应用每个Option函数到chart实例上。
	// 这允许调用者自定义配置Chart实例。
	//NewChart函数可以接受任意数量的配置函数，遍历出option代表着不同的Chart配置，并对每个元素（即每个配置函数）调用option(chart)，将每个配置应用到新创建的Chart实例上。
	for _, option := range options {
		option(chart)
	}

	// 尝试从staticFiles文件系统中读取chart.js文件。
	// 这是图表的JavaScript脚本，用于前端显示。
	chartJS, err := staticFiles.ReadFile("assets/chart.js")
	if err != nil {
		return nil, err // 如果读取失败，返回错误
	}

	// 将assets/chart.html模板文件解析到chart.indexHTML字段。
	// 这个HTML文件是图表的主页模板。
	//template.ParseFS，专门用于从文件系统读取并解析模板文件。它是Go 1.16版本引入的，允许直接从嵌入的文件系统中读取模板文件，这在处理静态文件（如HTML模板）时非常有用。
	//这行代码的作用就是从staticFiles这个文件系统中解析位于"assets/chart.html"路径的文件。解析成功后，这个文件作为HTML模板被加载并存储在chart实例的indexHTML字段中
	chart.indexHTML, err = template.ParseFS(staticFiles, "assets/chart.html")
	if err != nil {
		return nil, err // 如果解析失败，返回错误
	}

	// 使用esbuild库转译chartJS字符串，进行语法压缩和转换，以适配ES2015标准。
	// debug模式下关闭压缩功能，以便于调试。
	//这段代码通过esbuild对chart.js文件内容进行转译和压缩处理的主要作用是为了优化最终生成的JavaScript代码，使其更适合在生产环境中使用。
	// 减小文件体积：通过压缩语法、标识符和空白符，减少不必要的字符，比如空格、换行符和注释，以及缩短变量名和函数名，从而显著减小文件的体积。
	// 提高执行效率：压缩后的代码体积更小，可以更快地被浏览器解析和执行。
	// 兼容性：将代码转译为ES2015（ES6）标准，可以确保在各种现代浏览器上都能正常运行，即使原始代码使用了更高版本的ECMAScript语法特性。
	// 条件性压缩：根据chart.debug标志动态开启或关闭压缩，使得开发者在调试阶段可以查看更易读的代码，而在生产环境中则利用压缩后的代码以优化性能。
	transpileChartJS := api.Transform(string(chartJS), api.TransformOptions{
		Loader:            api.LoaderJS, // 指定了加载器类型为JavaScript。这告诉esbuild，输入的源代码是JavaScript代码，应该如何处理这种类型的文件。
		Target:            api.ES2015,   // 设定目标代码的ECMAScript版本为ES2015（也称为ES6）。这意味着esbuild将尝试将源代码转译为兼容ES2015标准的代码，以确保在支持该标准的环境中正常运行。
		MinifySyntax:      !chart.debug, // 根据debug状态决定是否压缩语法，当chart.debug为false时，启用语法压缩。语法压缩包括简化代码结构而不改变其功能，比如移除不必要的空格、换行和注释等，使得代码更紧凑。
		MinifyIdentifiers: !chart.debug, // 根据debug状态决定是否压缩标识符，当chart.debug为false时，启用标识符压缩。这通常涉及缩短变量名和函数名等，以减少代码体积。
		MinifyWhitespace:  !chart.debug, // 根据debug状态决定是否压缩空白符，当chart.debug为false时，启用空白符压缩。这意味着会移除代码中不必要的空格和换行符，进一步减少代码体积。
	})

	// 如果压缩文件的错误大于0  表示有错误 打印出错误
	if len(transpileChartJS.Errors) > 0 {
		return nil, fmt.Errorf("chart script failed with: %v", transpileChartJS.Errors)
	}

	// 将转译后的代码赋值给chart.scriptContent，以便于在前端使用。
	chart.scriptContent = string(transpileChartJS.Code)

	// 最终返回配置完成的chart实例和nil错误。
	return chart, nil
}
