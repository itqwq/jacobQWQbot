package exchange

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/xhit/go-str2duration/v2"

	"github.com/rodrigo-brito/ninjabot/model"
)

// 定义一个错误变量，用于表示数据不足的情况。
var ErrInsufficientData = errors.New("insufficient data")

// PairFeed结构体定义了一个用于处理和转换蜡烛图数据的交易对的配置信息。
// 包括交易对名称、CSV文件路径、数据的时间帧（如1m, 5m, 1h等），以及是否应用Heikin Ashi蜡烛图转换。
type PairFeed struct {
	Pair       string // 交易对名称，如"BTCUSDT"
	File       string // File字段就是用来指定CSV文件的位置的。这个字段应该包含CSV文件的完整路径
	Timeframe  string // Timeframe字段设置的时间间隔确实决定了蜡烛图数据更新的频率，如1d，就是一天更新一次。
	HeikinAshi bool   // 是否使用Heikin Ashi(平均k线图)样式的蜡烛图
}

// CSVFeed 结构体包含了所有PairFeed的映射，以及一个映射来存储每个交易对和时间帧对应的蜡烛图数据。
type CSVFeed struct {
	Feeds               map[string]PairFeed       // Feeds映射是CSVFeed结构体的一部分，它存储了关于每个交易对数据源的配置信息。以交易对名称(如btc)为键，然后CSVFeed调用里面的CSV文件，拿到蜡烛图数据
	CandlePairTimeFrame map[string][]model.Candle // 存储蜡烛图数据，csvFeed实例在读取CSV文件并将其中的数据转换成蜡烛图数据！，会将这些蜡烛图数据保存到它的CandlePairTimeFrame映射中。这个映射以一个由交易对名称和时间帧组成的字符串作为键（例如： "BTCUSDT--1h": btcCandles, // BTC/USDT，1小时时间帧的蜡烛图数据）
}

// AssetsInfo 方法接受一个交易对名称，返回该交易对的资产信息，包括基础资产、报价资产、最大价格、最大数量等。
func (c CSVFeed) AssetsInfo(pair string) model.AssetInfo {
	// 调用SplitAssetQuote函数（假设在别处定义）来分解交易对名称为基础资产和报价资产。
	asset, quote := SplitAssetQuote(pair)

	// 返回model.AssetInfo结构体实例，填充了基础资产、报价资产和其他相关信息。
	// 这里的值部分是硬编码的，如最大价格和数量设为float64的最大值，步长和精度设定为特定值。
	return model.AssetInfo{
		BaseAsset:          asset,           // 基础资产，如"BTC"
		QuoteAsset:         quote,           // 报价资产，如"USDT"
		MaxPrice:           math.MaxFloat64, // 最大价格，设置为float64的最大值
		MaxQuantity:        math.MaxFloat64, // 最大数量，同样设置为float64的最大值
		StepSize:           0.00000001,      // 步长，这里假设为1e-8
		TickSize:           0.00000001,      // 最小价格变动，假设为1e-8
		QuotePrecision:     8,               // 报价资产精度价格可以有的小数点后的最大位数。精度为8意味着报价（价格）可以精确到小数点后第八位1.12345678
		BaseAssetPrecision: 8,               // 基础资产精度，精度为8表示交易的数量可以精确到小数点后第八位，例如，你可以买卖0.12345678 BTC。
	}
}

// parseHeaders函数的目的是解析CSV文件中的表头（Header），并根据这些表头来确定每个重要字段（如时间、开盘价、收盘价等）在CSV中的索引位置。
// 函数返回三个值：一个映射（从表头名称映射到其在CSV文件中的索引位置）、一个包含了所有额外表头的切片、以及一个布尔值表示是否成功解析所有预定义表头。
func parseHeaders(headers []string) (index map[string]int, additional []string, ok bool) {
	// headerMap  就是 一个对照物希望传过来的参数有headerMap  的表头 如果没有 就把他放在additional盒子里面
	headerMap := map[string]int{
		"time": 0, "open": 1, "close": 2, "low": 3, "high": 4, "volume": 5,
	}

	// 个函数尝试将headers切片的第一个元素 表头的第一列名称如果被转化成整数 格式就是有错误，因为表头通常包含描述性的文本，如"Date", "Open", "Close"等
	_, err := strconv.Atoi(headers[0])
	if err == nil {
		// 如果没有错误，说明第一个表头可以被解析为数字，这不符合我们的预期，因此返回false。
		return headerMap, additional, false
	}

	// 遍历传入的表头，确定每个表头的实际索引位置。
	// 如果表头不在预定义的map中，则认为它是一个额外的表头，并将其添加到additional切片中。
	for index, h := range headers {
		if _, ok := headerMap[h]; !ok {
			additional = append(additional, h)
		}
		// 更新或设置headerMap中对应表头h的值为当前的索引index
		headerMap[h] = index
	}

	// 返回更新后的headerMap、找到的额外表头以及true表示成功解析预定义表头。
	return headerMap, additional, true
}

// NewCSVFeed 从CSV文件创建一个新的数据源，并根据目标时间框架对数据进行重采样。
// NewCSVFeed函数的目的是从一个或多个CSV文件中读取数据，可能对这些数据进行一些处理（如重采样），然后将处理后的数据封装在一个CSVFeed结构体中返回。
func NewCSVFeed(targetTimeframe string, feeds ...PairFeed) (*CSVFeed, error) {
	// 初始化CSVFeed实例，其中包括两个映射结构：Feeds 和 CandlePairTimeFrame。
	csvFeed := &CSVFeed{
		Feeds:               make(map[string]PairFeed),       // 存储PairFeed信息的映射。
		CandlePairTimeFrame: make(map[string][]model.Candle), // 存储各时间框架内蜡烛图数据的映射。
	}

	// 遍历所有传入的feeds。
	for _, feed := range feeds {
		// 将当前feed存储到Feeds映射中。
		csvFeed.Feeds[feed.Pair] = feed

		// 打开对应的CSV文件。
		csvFile, err := os.Open(feed.File)
		if err != nil {
			return nil, err
		}

		// 使用csv.NewReader读取并解析CSV文件中的所有行。然后拿到的数放在切片csvLines中
		csvLines, err := csv.NewReader(csvFile).ReadAll()
		if err != nil {
			return nil, err
		}

		// 准备解析CSV文件中的数据行为model.Candle结构体。
		var candles []model.Candle
		ha := model.NewHeikinAshi() // 创建一个HeikinAshi实例用于生成平均蜡烛图。

		// 解析CSV文件的表头，确定每个重要字段的索引位置，并检查是否有自定义的额外表头。如果有额外的会放在additionalHeaders里面headerMap 键是表头名称，值是表头对应的index
		headerMap, additionalHeaders, hasCustomHeaders := parseHeaders(csvLines[0])
		if hasCustomHeaders {
			// 。因为第一行表头行已经被解析了，我们已经知道了每个字段的名称和它们在CSV文件中的索引位置。这些信息足够我们处理文件中的数据行，因此表头行本身在这之后就不再需要了。
			csvLines = csvLines[1:]
		}

		// 遍历CSV文件的每一行，将其转换为model.Candle实例。
		for _, line := range csvLines {
			// 解析时间戳字段 变成整数。line是一个字符串切片，代表CSV文件中的每一行数据，
			timestamp, err := strconv.Atoi(line[headerMap["time"]])
			if err != nil {
				return nil, err
			}

			// 创建并填充Candle实例的字段。
			candle := model.Candle{
				// 将一个表示秒数的Unix时间戳（timestamp）转换为一个time.Time类型的值，UTC()世界标注时间
				Time:      time.Unix(int64(timestamp), 0).UTC(),
				UpdatedAt: time.Unix(int64(timestamp), 0).UTC(),
				Pair:      feed.Pair,
				Complete:  true,
			}

			// 解析并设置蜡烛图的其他属性（开盘价、收盘价、最低价、最高价、成交量）。
			candle.Open, err = strconv.ParseFloat(line[headerMap["open"]], 64)
			if err != nil {
				return nil, err
			}
			// 重复上述过程解析close, low, high, volume等字段。

			// 如果有自定义的额外表头，将这些额外信息添加到蜡烛图的Metadata中。
			if hasCustomHeaders {
				candle.Metadata = make(map[string]float64) //这个映射用于存储额外表头及其对应的数据值。
				for _, header := range additionalHeaders {
					// 这行代码尝试将某个字段的字符串值转换为一个64位的浮点数，并将这个数值存储在candle.Metadata映射中键是header 值是浮点数
					candle.Metadata[header], err = strconv.ParseFloat(line[headerMap[header]], 64)
					if err != nil {
						return nil, err
					}
				}
			}

			// 如果feed配置了HeikinAshi转换，则对蜡烛图进行相应的转换。
			if feed.HeikinAshi {
				candle = candle.ToHeikinAshi(ha)
			}

			// 将处理好的蜡烛图添加到列表中。
			candles = append(candles, candle)
		}

		// 将解析好的蜡烛图数据存储到CandlePairTimeFrame映射中，键为对应的货币对和时间框架。
		csvFeed.CandlePairTimeFrame[csvFeed.feedTimeframeKey(feed.Pair, feed.Timeframe)] = candles

		// 根据目标时间框架对蜡烛图数据进行重采样。
		err = csvFeed.resample(feed.Pair, feed.Timeframe, targetTimeframe)
		if err != nil {
			return nil, err
		}
	}

	// 返回初始化完成且填充了数据的CSVFeed实例。
	return csvFeed, nil

}

// feedTimeframeKey 是 CSVFeed 类型的一个方法，feedTimeframeKey 意思是这个方法是把交易对还有时间框架连起来的唯一标识符。 如"BTC/USD--1h" 表示 可以包含每小时的开盘价、收盘价、最高价、最低价和成交量等信息
func (c CSVFeed) feedTimeframeKey(pair, timeframe string) string {
	// 使用 fmt.Sprintf 函数将 pair 和 timeframe 参数格式化为一个字符串，
	// 中间用 "--" 分隔。这个格式化的字符串作为货币对和时间框架的唯一标识符。
	// 例如，如果 pair 是 "BTC/USD"，timeframe 是 "1h"，则返回的字符串将是 "BTC/USD--1h"。
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// 这个方法功能还没有实现等待开发
func (c CSVFeed) LastQuote(_ context.Context, _ string) (float64, error) {
	return 0, errors.New("invalid operation")
}

// Limit 方法用于限制蜡烛图数据的时间范围。它接收一个时间段作为参数，并且会对每个货币对的蜡烛图数据进行处理，将超出指定时间段的数据移除。
// 我们有一份包含一年内某个货币对的蜡烛图数据，每个蜡烛图代表一天的价格变动。现在我们想要获取最近一个月内的数据来进行分析。这时候就可以使用 Limit 方法，传入一个时间段，比如30天，然后它会帮助我们筛选出最近30天内的蜡烛图数据，以
func (c *CSVFeed) Limit(duration time.Duration) *CSVFeed {
	// 遍历每个货币对及其对应的蜡烛图数据
	for pair, candles := range c.CandlePairTimeFrame {
		// candles[len(candles)-1] 最后蜡烛图反应最新的数据， Time.Add(-duration) 设置一个新的起点，原来时间减去传入的时间，表示往回退了这个时间表示开始时间，所以就只含有duration 时间的数
		start := candles[len(candles)-1].Time.Add(-duration)

		// 使用过滤函数 Filter 对蜡烛图数据进行筛选，保留在指定时间段内的数据，它接收两个参数：一个是要过滤的数据集，这个函数定义了过滤的条件
		c.CandlePairTimeFrame[pair] = lo.Filter(candles, func(candle model.Candle, _ int) bool {
			// 检查蜡烛图的时间戳是否晚于（即在之后）start 时间点，专门设计来只保留在给定的时间段 duration 内的数据，即从 start 时间点到最后一个蜡烛图时间点之间的数据。
			return candle.Time.After(start)
		})
	}
	return c
}

// isFistCandlePeriod 函数检查给定的时间点t是否为从源时间框架到目标时间框架的重采样中第一个周期的开始。
func isFistCandlePeriod(t time.Time, fromTimeframe, targetTimeframe string) (bool, error) {
	// 将源时间框架字符串转换为时间间隔（Duration）。如fromTimeframe是"1h"，表示1小时。经过转换，fromDuration变成了3,600,000,000,000纳秒
	fromDuration, err := str2duration.ParseDuration(fromTimeframe)
	if err != nil {
		// 如果转换失败，则返回错误。
		return false, err
	}

	// 计算前一个周期的开始时间。这是通过从给定时间点t减去源时间框架的持续时间来实现的。
	prev := t.Add(-fromDuration).UTC()

	// 调用isLastCandlePeriod函数，检查prev是否位于目标时间框架周期的最后。
	// 如果目标时间框架是1h，那么一天分为24个周期。假设前一个时间点prev是2024年3月15日 23:00 UTC，
	// 这表示它是当天最后一个小时周期的开始。因此，如果prev在一个小时周期的最后，
	// 则t（紧随prev的时间点）标志着新的周期的开始。这里，我们通过检查prev来判断t是否是新周期的开始。
	return isLastCandlePeriod(prev, fromTimeframe, targetTimeframe)
}

// iisLastCandlePeriod：通过查看当前时间点之后的时间来判断这个点是否为当前周期的末尾。
// 这个方法结束就意味着图的完整 那么也意味着下个周期的开始
func isLastCandlePeriod(t time.Time, fromTimeframe, targetTimeframe string) (bool, error) {
	// 如果源时间框架等于目标时间框架，那么任何给定时间点t都可以被认为是这个时间框架周期的结束。 就是假设而已没有任何根据
	if fromTimeframe == targetTimeframe {
		return true, nil
	}

	// 解析源时间框架的持续时间。
	fromDuration, err := str2duration.ParseDuration(fromTimeframe)
	if err != nil {
		return false, err // 如果解析失败，则返回错误。
	}

	// 计算 t 的下一个时间点。这里的“下一个时间点”可以理解为下一个时间周期的开始点。例如，如果t表示2024年3月15日 14:00 UTC，并且fromTimeframe是"1h"（fromDuration因此为1小时），那么计算得到的next将会是2024年3月15日 15:00 UTC。这意味着，基于1小时的时间框架，
	next := t.Add(fromDuration).UTC()

	// 根据目标时间框架的要求，检查下一个时间点是否符合周期要求。无论是1分钟、5分钟、1小时等周期的开始，关键在于对应时间单位（分钟或小时）除以周期长度（如1、5、10等）的余数为0。这表明当前时间点可以被周期长度整除，从而标志着新的时间段的开始。
	// 这段代码的目的正是检查下一个时间点（next）是否恰好位于目标时间框架指定周期的开始位置。这个判断基于不同的时间框架（从一分钟到一周）
	switch targetTimeframe {
	case "1m": //一分钟
		return next.Second()%60 == 0, nil
	case "5m":
		return next.Minute()%5 == 0, nil //表示next的分钟数正好是5的倍数，也就是说，next代表的时间点恰好在一个5分钟周期的开始。
	case "10m":
		return next.Minute()%10 == 0, nil
	case "15m":
		return next.Minute()%15 == 0, nil
	case "30m":
		return next.Minute()%30 == 0, nil
	case "1h":
		return next.Minute()%60 == 0, nil
	case "2h":
		return next.Minute() == 0 && next.Hour()%2 == 0, nil
	case "4h":
		return next.Minute() == 0 && next.Hour()%4 == 0, nil
	case "12h":
		return next.Minute() == 0 && next.Hour()%12 == 0, nil
	case "1d":
		return next.Minute() == 0 && next.Hour()%24 == 0, nil
	case "1w":
		// 如果next是周日的午夜（0点0分），那么这个表达式会返回true, nil，表示next是一周的开始。
		return next.Minute() == 0 && next.Hour()%24 == 0 && next.Weekday() == time.Sunday, nil
	}

	// 当isLastCandlePeriod函数遇到一个不被识别或者无效的targetTimeframe时，确实会直接返回false和一个错误，指示传入的时间框架无效
	return false, fmt.Errorf("invalid timeframe: %s", targetTimeframe)
}

// resample 方法将特定货币对的蜡烛图数据从源时间框架重新采样到目标时间框架。
// 如将五个一小时的数据合成一个五小时的数据概括如下：开盘价取自第一小时，收盘价来自第五小时，最高价和最低价分别是五小时里的最高和最低，成交量则是累加的总和。
// 参数交易对，原始时间间隔，目标时间间隔
func (c *CSVFeed) resample(pair, sourceTimeframe, targetTimeframe string) error {
	// 使用货币对和时间框架生成源和目标的键。这两个键帮助我们区分和访问不同时间框架下的数据。
	sourceKey := c.feedTimeframeKey(pair, sourceTimeframe)
	targetKey := c.feedTimeframeKey(pair, targetTimeframe)

	// 找到源时间框架数据中的第一个符合目标时间框架开始周期的蜡烛图。
	var i int
	// 这是一个for循环，它从先前设置的索引i开始，一直增加到sourceKey对应的数据集合的长度减一的位置。
	//len(c.CandlePairTimeFrame[sourceKey])计算的是源数据集合的大小，即有多少个蜡烛图
	// 这段代码的目的是在数据集中找到第一个符合目标时间框架开始条件的蜡烛图。一旦找到这个蜡烛图，就会从这个点开始进行后续的数据处理或分析，比如重新采样或聚合数据以适应目标时间框架。 这个[sourceKey][i].Time, 是不是 在	targetTimeframe 目标时间的开始
	for ; i < len(c.CandlePairTimeFrame[sourceKey]); i++ {
		if ok, err := isFistCandlePeriod(c.CandlePairTimeFrame[sourceKey][i].Time, sourceTimeframe,
			targetTimeframe); err != nil {
			return err // 如果检查过程中出现错误，则返回错误。
		} else if ok {
			break // 找到了符合条件的第一个蜡烛图，跳出循环。
		}
	}

	// 初始化一个用于存放重新采样后的蜡烛图数据的切片。
	//在这段代码中，合并是指将当前蜡烛图与前一个蜡烛图进行合并，以创建一个新的蜡烛图，其中包含两个蜡烛图的信息。
	candles := make([]model.Candle, 0)
	for ; i < len(c.CandlePairTimeFrame[sourceKey]); i++ {
		candle := c.CandlePairTimeFrame[sourceKey][i] // 当前处理的蜡烛图。
		// 确认蜡烛图是否完整。这个方法结束就意味着图的完整 那么也意味着下个周期的开始
		if last, err := isLastCandlePeriod(candle.Time, sourceTimeframe, targetTimeframe); err != nil {
			return err // 如果检查过程中出现错误，则返回错误。
		} else if last {
			candle.Complete = true // 标记为完整周期的蜡烛图。
		} else {
			candle.Complete = false // ，当isLastCandlePeriod函数返回false时，这表示当前蜡烛图的时间点不是目标时间框架的周期结束点。在这种情况下，蜡烛图被标记为不完整，
		}

		// 如果当前蜡烛图不是新周期的第一个，则需要与上一个蜡烛图合并。
		//最后一个索引”（lastIndex）指的是在当前操作之前candles切片中的最后一个蜡烛图的位置。
		lastIndex := len(candles) - 1
		// 这行检查上一次添加到candles数组中的蜡烛图是否标记为不完整
		if lastIndex >= 0 && !candles[lastIndex].Complete {
			// 合并逻辑：保持开始时间和开盘价不变，计算最高价、最低价和总成交量。
			candle.Time = candles[lastIndex].Time                        //两个蜡烛图合并的时候上一个蜡烛图的开始时间就是合并后的开盘价
			candle.Open = candles[lastIndex].Open                        //两个蜡烛图合并的时候上一个蜡烛图的开盘价就是合并后的开盘价
			candle.High = math.Max(candles[lastIndex].High, candle.High) //对比上个蜡烛图和现在的最高价，然后取最高价
			candle.Low = math.Min(candles[lastIndex].Low, candle.Low)    //对比上个蜡烛图和现在的最低价，然后取最低价
			candle.Volume += candles[lastIndex].Volume                   //这行代码将上一个蜡烛图的成交量加到当前处理的蜡烛图的成交量上，从而得到合并后的总成交量。
		}
		// 将处理好的蜡烛图添加到切片中。
		candles = append(candles, candle)
	}

	// 如果最后一个蜡烛图不是完整的，则将其移除。
	if !candles[len(candles)-1].Complete {
		//这个子切片包含了从索引 0 到倒数第二个元素的所有元素，即移除了最后一个元素。
		candles = candles[:len(candles)-1]
	}

	// 将重新采样后的蜡烛图数据存储到目标键下。
	c.CandlePairTimeFrame[targetKey] = candles

	return nil // 成功完成重新采样，无错误返回。
}

func (c CSVFeed) CandlesByPeriod(_ context.Context, pair, timeframe string,
	start, end time.Time) ([]model.Candle, error) {

	key := c.feedTimeframeKey(pair, timeframe)
	candles := make([]model.Candle, 0)
	for _, candle := range c.CandlePairTimeFrame[key] {
		if candle.Time.Before(start) || candle.Time.After(end) {
			continue
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

// CandlesByLimit 从 CSVFeed 中获取指定货币对和时间框架的蜡烛图数据，限制返回的蜡烛图数量为 limit。
func (c *CSVFeed) CandlesByLimit(_ context.Context, pair, timeframe string, limit int) ([]model.Candle, error) {
	var result []model.Candle                    // 初始化结果切片，用于存储蜡烛图数据。
	key := c.feedTimeframeKey(pair, timeframe)   // 根据货币对和时间框架生成键值。
	if len(c.CandlePairTimeFrame[key]) < limit { // 检查指定键值对应的蜡烛图数据是否小于指定的限制数量。
		return nil, fmt.Errorf("%w: %s", ErrInsufficientData, pair) // 当蜡烛图数据数量少于限制时，会返回错误，这是因为该函数的预期行为是返回指定数量的蜡烛图数据。
	}
	// 从 0 到 limit-1 的蜡烛图，不包括索引为 limit 的蜡烛图。 保存在result 包括索引为 limit 开始直到最后一个蜡烛图的所有数据。 保存在c.CandlePairTimeFrame[key]
	result, c.CandlePairTimeFrame[key] = c.CandlePairTimeFrame[key][:limit], c.CandlePairTimeFrame[key][limit:]
	return result, nil // 返回提取的蜡烛图数据。
}

// CandlesSubscription方法用于订阅蜡烛图数据 返回两个通道：一个用于发送蜡烛图数据，另一个用于发送错误信息。
func (c CSVFeed) CandlesSubscription(_ context.Context, pair, timeframe string) (chan model.Candle, chan error) {
	// 创建两个通道，一个用于发送蜡烛图数据，另一个用于发送错误信息。
	ccandle := make(chan model.Candle) // 用于发送蜡烛图数据的通道
	cerr := make(chan error)           // 用于发送错误信息的通道

	// 根据货币对和时间框架生成键，以便从数据集中获取相应的蜡烛图数据。
	key := c.feedTimeframeKey(pair, timeframe)

	// 启动一个 Go 协程，从数据集中逐个发送蜡烛图数据到通道中。
	go func() {
		for _, candle := range c.CandlePairTimeFrame[key] {
			ccandle <- candle // 将蜡烛图数据发送到 ccandle 通道中
		}
		close(ccandle) // 关闭发送蜡烛图数据的通道
		close(cerr)    // 关闭发送错误信息的通道
	}()

	// 返回蜡烛图数据通道和错误信息通道。
	return ccandle, cerr
}
