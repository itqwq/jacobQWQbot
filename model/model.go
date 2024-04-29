// 定义模型包
package model

// 导入必要的包
import (
	"fmt"     // 用于格式化输出
	"math"    // 提供基本的数学函数
	"strconv" // 提供字符串与基本数据类型的转换
	"time"    // 提供时间相关的函数和方法
)

// TelegramSettings 定义了Telegram通知的设置
type TelegramSettings struct {
	Enabled bool   // 是否启用Telegram通知
	Token   string // Telegram bot的Token
	Users   []int  // 接收通知的用户ID列表
}

// Settings 定义了整个应用的设置
// 这个Settings结构体用于定义应用的配置，包括支持的交易对列表和Telegram通知设置。
type Settings struct {
	Pairs    []string         // 交易对列表
	Telegram TelegramSettings // Telegram通知的设置
}

// Balance 定义了资产余额的结构
type Balance struct {
	Asset    string  // 资产标识 如ETH
	Free     float64 // 可用余额 标识资产数量
	Lock     float64 // 锁定余额这可能是因为你用这部分BTC作为了某个未平仓合约的保证金，或者你已经下了一个尚未成交的卖出订单。
	Leverage float64 // 杠杆倍数
}

// AssetInfo 定义了资产信息的结构
type AssetInfo struct {
	BaseAsset  string // 基础资产
	QuoteAsset string // BTC/USD，其中BTC是基础资产，USD是报价资产如果BTC/USD的价格为50000，那么这意味着你需要支付50000美元才能购买1个比特币。

	MinPrice    float64 // 最小价格
	MaxPrice    float64 // 最大价格
	MinQuantity float64 // 最小数量
	MaxQuantity float64 // 最大数量
	StepSize    float64 // 步长大小 就比如我设置0.01 然后订单只能是0.01，0.02，0.03 就是他的倍数
	TickSize    float64 // 价格变动的最小单位如果这个交易对的 TickSize 设置为 0.01 美元，那么价格可以在 100、100.01、100.02、100.03

	QuotePrecision     int // 报价精度QuotePrecision 关注于交易价格的精度。
	BaseAssetPrecision int // 基础资产精度BaseAssetPrecision 关注于交易数量的精度
}

// Dataframe 定义了数据帧的结构，用于存储和处理时间序列数据，可以根据每个时间点给出这个交易对的数据，如最高价，最低价，交易量等
type Dataframe struct {
	Pair string // 交易对

	Close  Series[float64] // 收盘价序列
	Open   Series[float64] // 开盘价序列
	High   Series[float64] // 最高价序列
	Low    Series[float64] // 最低价序列
	Volume Series[float64] // 成交量序列

	Time       []time.Time // 时间戳序列
	LastUpdate time.Time   // 最后更新时间

	// 自定义用户元数据
	Metadata map[string]Series[float64]
}

// Sample 方法用于从Dataframe中抽取最近的N个数据点作为一个新的Dataframe
func (df Dataframe) Sample(positions int) Dataframe {
	size := len(df.Time)
	start := size - positions
	if start <= 0 {
		return df
	}

	sample := Dataframe{
		Pair:       df.Pair,
		Close:      df.Close.LastValues(positions),
		Open:       df.Open.LastValues(positions),
		High:       df.High.LastValues(positions),
		Low:        df.Low.LastValues(positions),
		Volume:     df.Volume.LastValues(positions),
		Time:       df.Time[start:],
		LastUpdate: df.LastUpdate,
		Metadata:   make(map[string]Series[float64]),
	}

	for key := range df.Metadata {
		sample.Metadata[key] = df.Metadata[key].LastValues(positions)
	}

	return sample
}

// Candle 定义了K线的结构
type Candle struct {
	Pair      string    // 交易对
	Time      time.Time // 时间戳
	UpdatedAt time.Time // 更新时间
	Open      float64   // 开盘价
	Close     float64   // 收盘价
	Low       float64   // 最低价
	High      float64   // 最高价
	Volume    float64   // 成交量
	Complete  bool      // 是否完成

	// 从CSV输入中附加的额外列
	Metadata map[string]float64
}

// Empty 方法用于判断一个K线是否为空
func (c Candle) Empty() bool {
	return c.Pair == "" && c.Close == 0 && c.Open == 0 && c.Volume == 0
}

// HeikinAshi 定义了平均K线(Heikin Ashi)的结构
type HeikinAshi struct {
	PreviousHACandle Candle // 前一个平均K线
}

// NewHeikinAshi 函数用于创建一个新的HeikinAshi实例
func NewHeikinAshi() *HeikinAshi {
	return &HeikinAshi{}
}

// ToSlice 方法将Candle的数据转换成字符串切片，通常用于文件输出
func (c Candle) ToSlice(precision int) []string {
	return []string{
		fmt.Sprintf("%d", c.Time.Unix()),                  // 时间戳
		strconv.FormatFloat(c.Open, 'f', precision, 64),   // 开盘价
		strconv.FormatFloat(c.Close, 'f', precision, 64),  // 收盘价
		strconv.FormatFloat(c.Low, 'f', precision, 64),    // 最低价
		strconv.FormatFloat(c.High, 'f', precision, 64),   // 最高价
		strconv.FormatFloat(c.Volume, 'f', precision, 64), // 成交量
	}
}

// ToHeikinAshi 方法将普通K线转换为平均K线（Heikin Ashi）
func (c Candle) ToHeikinAshi(ha *HeikinAshi) Candle {
	// // CalculateHeikinAshi 方法用于计算并返回一个平均K线
	haCandle := ha.CalculateHeikinAshi(c)

	return Candle{
		Pair:      c.Pair,
		Open:      haCandle.Open,
		High:      haCandle.High,
		Low:       haCandle.Low,
		Close:     haCandle.Close,
		Volume:    c.Volume,
		Complete:  c.Complete,
		Time:      c.Time,
		UpdatedAt: c.UpdatedAt,
	}
}

// Less 方法用于比较两个Candle的时间，用于排序
func (c Candle) Less(j Item) bool {
	diff := j.(Candle).Time.Sub(c.Time)
	if diff < 0 {
		return false
	}
	if diff > 0 {
		return true
	}

	diff = j.(Candle).UpdatedAt.Sub(c.UpdatedAt)
	if diff < 0 {
		return false
	}
	if diff > 0 {
		return true
	}

	return c.Pair < j.(Candle).Pair
}

// Account 定义了账户的结构
type Account struct {
	Balances []Balance // 账户的资产余额列表
}

// Balance 方法用于从账户中获取指定的基础资产和报价资产的余额信息。
// 它接收两个字符串参数：assetTick 和 quoteTick，分别表示基础资产和报价资产的标识符。
// assetTick和quoteTick分别代表交易中的基础资产如BTC 和报价资产的标识符，如USD。
func (a Account) Balance(assetTick, quoteTick string) (Balance, Balance) {
	// 初始化两个 Balance 类型的变量用于存储找到的基础资产和报价资产的余额信息。
	var assetBalance, quoteBalance Balance
	// 初始化两个布尔类型的变量，用于标记是否已经找到了基础资产和报价资产的余额信息。
	var isSetAsset, isSetQuote bool

	// 遍历账户中的所有余额信息。
	for _, balance := range a.Balances {
		switch balance.Asset {
		case assetTick: // 如果当前余额信息的资产标识符匹配基础资产标识符
			assetBalance = balance // 将这个余额信息存储为基础资产的余额
			isSetAsset = true      // 标记已找到基础资产的余额信息
		case quoteTick: // 如果当前余额信息的资产标识符匹配报价资产标识符
			quoteBalance = balance // 将这个余额信息存储为报价资产的余额
			isSetQuote = true      // 标记已找到报价资产的余额信息
		}

		// 如果已经找到了基础资产和报价资产的余额信息，则提前终止循环。
		if isSetAsset && isSetQuote {
			break
		}
	}

	// 返回找到的基础资产和报价资产的余额信息。
	return assetBalance, quoteBalance
}

// Equity 方法计算并返回账户的总权益
func (a Account) Equity() float64 {
	var total float64

	for _, balance := range a.Balances {
		total += balance.Free
		total += balance.Lock
	}

	return total
}

// CalculateHeikinAshi 方法用于计算并返回一个平均K线
func (ha *HeikinAshi) CalculateHeikinAshi(c Candle) Candle {
	var hkCandle Candle

	openValue := ha.PreviousHACandle.Open
	closeValue := ha.PreviousHACandle.Close

	// 如果是第一个平均K线，则使用当前K线的数据
	if ha.PreviousHACandle.Empty() {
		openValue = c.Open
		closeValue = c.Close
	}

	hkCandle.Open = (openValue + closeValue) / 2                              // 计算开盘价
	hkCandle.Close = (c.Open + c.High + c.Low + c.Close) / 4                  // 计算收盘价
	hkCandle.High = math.Max(c.High, math.Max(hkCandle.Open, hkCandle.Close)) // 计算最高价
	hkCandle.Low = math.Min(c.Low, math.Min(hkCandle.Open, hkCandle.Close))   // 计算最低价
	ha.PreviousHACandle = hkCandle                                            // 更新前一个平均K线

	return hkCandle
}
