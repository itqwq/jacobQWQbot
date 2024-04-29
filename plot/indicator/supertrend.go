package indicator // 定义了一个包名为indicator，用于存放技术分析指标的实现

import (
	"fmt"  // 导入fmt包，用于格式化字符串
	"time" // 导入time包，用于处理时间相关的功能

	"github.com/rodrigo-brito/ninjabot/model" // 导入ninjabot的model包，用于访问交易数据结构
	"github.com/rodrigo-brito/ninjabot/plot"  // 导入ninjabot的plot包，用于绘制指标图表

	"github.com/markcheno/go-talib" // 导入go-talib包，用于计算各种技术指标
)

// Spertrend函数创建并返回一个SuperTrend指标对象
// SuperTrend指标是一种流行的趋势跟踪工具，它结合了价格的移动方向和波动率两个因素来确定市场趋势的变化。这个指标特别适用于确定当前趋势的持续性和反转信号，因此被许多交易者用于股票、外汇、期货和其他金融市场的技术分析中。
// SuperTrend在k线图的上下方当SuperTrend线显示在价格下方，并且从红色（或其他表示下跌趋势的颜色）转变为绿色（或其他表示上升趋势的颜色），这表明趋势可能从下跌转为上升。此时，SuperTrend线提供了一个买入信号。相反，当SuperTrend线显示在价格上方，并且从绿色转变为红色，这表明趋势可能从上升转为下跌。此时，SuperTrend线提供了一个卖出信号。
func Spertrend(period int, factor float64, color string) plot.Indicator {
	return &supertrend{
		Period: period, // 指定计算SuperTrend时使用的周期
		Factor: factor, // 指定计算时使用的因子，调整指标灵敏度
		Color:  color,  // 指定绘制指标线条时使用的颜色
	}
}

// supertrend结构体定义了SuperTrend指标所需的基本属性
// Factor因子确实是用来调整指标的灵敏度，Factor因子较小，让SuperTrend指标对价格的微小变动就做出反应，快速调整其趋势信号，指标会更频繁地进出市场，可能会因为市场的轻微变化就做出反应。
type supertrend struct {
	Period         int                   // 计算指标所需的周期长度
	Factor         float64               // 计算指标时使用的因子
	Color          string                // 绘图时使用的颜色
	Close          model.Series[float64] // 存储收盘价
	BasicUpperBand model.Series[float64] // 基本上带
	FinalUpperBand model.Series[float64] // 最终上带
	BasicLowerBand model.Series[float64] // 基本下带
	FinalLowerBand model.Series[float64] // 最终下带
	SuperTrend     model.Series[float64] // SuperTrend指标值
	Time           []time.Time           // 对应每个指标值的时间点
}

// Warmup方法返回计算该指标所需的最小数据点数，等于指定的周期
func (s supertrend) Warmup() int {
	return s.Period
}

// Name方法返回该指标的名称，包含周期和因子
func (s supertrend) Name() string {
	return fmt.Sprintf("SuperTrend(%d,%.1f)", s.Period, s.Factor)
}

// Overlay方法指示该指标是否应该覆盖在主图表上，对于SuperTrend，覆盖在价格图表上，所以返回true
func (s supertrend) Overlay() bool {
	return true
}

// Load方法负责加载数据帧并计算SuperTrend指标
func (s *supertrend) Load(df *model.Dataframe) {
	// 如果数据帧中的时间长度小于指定的周期，无法计算，直接返回
	if len(df.Time) < s.Period {
		return
	}

	// 使用ATR（平均真实范围）指标来计算波动性，作为SuperTrend计算的基础
	atr := talib.Atr(df.High, df.Low, df.Close, s.Period)

	// 初始化用于存储计算结果的切片
	s.BasicUpperBand = make([]float64, len(atr))
	s.BasicLowerBand = make([]float64, len(atr))
	s.FinalUpperBand = make([]float64, len(atr))
	s.FinalLowerBand = make([]float64, len(atr))
	s.SuperTrend = make([]float64, len(atr))

	// 循环遍历每个时间点，计算SuperTrend
	for i := 1; i < len(s.BasicLowerBand); i++ {
		// 计算基本上轨和下轨，基于中点价格加减ATR乘以一个因子
		s.BasicUpperBand[i] = (df.High[i]+df.Low[i])/2.0 + atr[i]*s.Factor
		s.BasicLowerBand[i] = (df.High[i]+df.Low[i])/2.0 - atr[i]*s.Factor

		// 计算最终上轨。如果当前基本上轨低于前一期的最终上轨，或前一期收盘价高于前一期最终上轨，
		// 当前期的最终上轨就是当前基本上轨，否则沿用前一期的最终上轨
		//由于没有前一周期的数据可以参考，我们直接将BasicUpperBand的值赋给FinalUpperBand。这个是初始设置，确保了序列有一个起点。
		if i == 0 {
			s.FinalUpperBand[i] = s.BasicUpperBand[i]
			//如果当前周期的BasicUpperBand值低于前一周期的FinalUpperBand值或者，如果前一周期的收盘价高于前一周期的FinalUpperBand值如果任一条件满足，当前周期的FinalUpperBand将被设置为当前周期的BasicUpperBand值。
		} else if s.BasicUpperBand[i] < s.FinalUpperBand[i-1] ||
			df.Close[i-1] > s.FinalUpperBand[i-1] {
			s.FinalUpperBand[i] = s.BasicUpperBand[i]
		} else {
			s.FinalUpperBand[i] = s.FinalUpperBand[i-1]
		}

		// 计算最终下轨，逻辑与最终上轨类似，但方向相反
		if i == 0 || s.BasicLowerBand[i] > s.FinalLowerBand[i-1] ||
			df.Close[i-1] < s.FinalLowerBand[i-1] {
			s.FinalLowerBand[i] = s.BasicLowerBand[i]
		} else {
			s.FinalLowerBand[i] = s.FinalLowerBand[i-1]
		}

		// 根据前一期的SuperTrend值和当前的收盘价确定本期的SuperTrend值
		if i == 0 || s.FinalUpperBand[i-1] == s.SuperTrend[i-1] {
			if df.Close[i] > s.FinalUpperBand[i] {
				s.SuperTrend[i] = s.FinalLowerBand[i]
			} else {
				s.SuperTrend[i] = s.FinalUpperBand[i]
			}
		} else {
			if df.Close[i] < s.FinalLowerBand[i] {
				s.SuperTrend[i] = s.FinalUpperBand[i]
			} else {
				s.SuperTrend[i] = s.FinalLowerBand[i]
			}
		}
	}

	// 将指标的计算结果从开始的周期裁剪，以便与时间序列对齐
	s.Time = df.Time[s.Period:]
	s.SuperTrend = s.SuperTrend[s.Period:]
}

func (s supertrend) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Style:  "scatter",
			Color:  s.Color,
			Values: s.SuperTrend,
			Time:   s.Time,
		},
	}
}
