package indicator

// 引入必要的包
import (
	"fmt"  // 用于格式化字符串
	"time" // 用于处理时间

	"github.com/rodrigo-brito/ninjabot/model" // ninjabot库中的模型，用于处理交易数据
	"github.com/rodrigo-brito/ninjabot/plot"  // ninjabot库的绘图部分，用于指标的图表展示

	"github.com/markcheno/go-talib" // TA-Lib库，用于计算各种技术指标
)

// BollingerBands函数创建并返回一个用于计算布林带指标的结构体实例
// 价格相对于中带的位置也可以用来判断市场的趋势方向：价格持续在中带之上可能表示上升趋势，
// 而价格持续在中带之下可能表示下降趋势。当价格突破上带时，这确实可能表示市场强劲和买盘压力增大，尤其是在上升趋势中。然而，这也可能被视为超买信号尤其是如果价格在没有回调的情况下迅速并大幅突破上带。在某些情况下，这可能预示着即将到来的价格回调或反转，而不是持续上涨。
// 当价格突破下带时，这可能表示卖压加大和市场弱势，特别是在下降趋势中。但这也可能被看作是超卖信号，可能预示着即将到来的反弹或价格反转，而不总是意味着继续下降。
func BollingerBands(period int, stdDeviation float64, upDnBandColor, midBandColor string) plot.Indicator {
	// 初始化布林带结构体，设置周期、标准偏差和带的颜色
	return &bollingerBands{
		Period:        period,        // 布林带计算的周期
		StdDeviation:  stdDeviation,  // 用于计算上下带宽度的标准偏差倍数
		UpDnBandColor: upDnBandColor, // 上下带的颜色
		MidBandColor:  midBandColor,  // 中带的颜色
	}
}

// bollingerBands结构体定义了布林带指标的配置和计算结果
// StdDeviation决定了上带和下带距离中带的宽度，它直接影响布林带的宽窄。 意思就是tdDeviation的值越大，上带和下带距离中带的宽度，越宽布林带宽度越宽：表示市场波动性较大。这意味着价格在短期内有较大的波动幅度，可能因为某些信息或事件导致市场参与者的看法分歧较大。布林带宽度越窄：表示市场波动性较小。这意味着价格变动较为温和，市场处于相对平静状态。布林带指的就是上带下带中带
type bollingerBands struct {
	Period        int                   // 计算中带的移动平均的周期
	StdDeviation  float64               // 上下带距离中带的标准偏差倍数
	UpDnBandColor string                // 上下带的颜色
	MidBandColor  string                // 中带的颜色
	UpperBand     model.Series[float64] // 上带值
	MiddleBand    model.Series[float64] // 中带值
	LowerBand     model.Series[float64] // 下带值
	Time          []time.Time           // 对应的时间序列
}

// Warmup方法返回指标计算前需要预热的数据长度，等同于布林带的周期
func (bb bollingerBands) Warmup() int {
	return bb.Period
}

// Name方法返回指标的名称和配置的参数
func (bb bollingerBands) Name() string {
	return fmt.Sprintf("BB(%d, %.2f)", bb.Period, bb.StdDeviation)
}

// Overlay方法指示布林带指标需要覆盖在价格图上
func (bb bollingerBands) Overlay() bool {
	return true
}

// Load方法负责加载数据帧并使用TA-Lib计算布林带的三条线（上带、中带、下带）
func (bb *bollingerBands) Load(dataframe *model.Dataframe) {
	// 若数据点不足以计算布林带，则直接返回
	if len(dataframe.Time) < bb.Period {
		return
	}

	// 调用TA-Lib函数计算布林带值
	//bb.StdDeviation参数出现了两次，这是因为TA-Lib的BBands函数允许分别为上带和下带指定不同的标准偏差倍数，尽管在大多数传统布林带的应用中，上带和下带使用相同的标准偏差倍数。
	upper, mid, lower := talib.BBands(dataframe.Close, bb.Period, bb.StdDeviation, bb.StdDeviation, talib.EMA)
	// 存储计算结果，并裁剪掉不足周期的数据点
	bb.UpperBand, bb.MiddleBand, bb.LowerBand = upper[bb.Period:], mid[bb.Period:], lower[bb.Period:]

	// 时间序列调整为与计算后的数据对应
	bb.Time = dataframe.Time[bb.Period:]
}

// Metrics方法定义了如何在图表上展示布林带的计算结果
func (bb bollingerBands) Metrics() []plot.IndicatorMetric {
	// 返回三条线的绘图配置，包括颜色和样式
	return []plot.IndicatorMetric{
		{
			Style:  "line",           // 线条样式
			Color:  bb.UpDnBandColor, // 上下带颜色
			Values: bb.UpperBand,     // 上
			Time:   bb.Time,
		},
		{
			Style:  "line",          // 线条样式
			Color:  bb.MidBandColor, // 中带颜色
			Values: bb.MiddleBand,   // 中带的值
			Time:   bb.Time,         // 对应的时间序列
		},
		{
			Style:  "line",           // 线条样式
			Color:  bb.UpDnBandColor, // 上下带颜色
			Values: bb.LowerBand,     // 下带的值
			Time:   bb.Time,          // 对应的时间序列
		},
	}
}
