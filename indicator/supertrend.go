// Package indicator 提供了金融市场技术指标的计算方法。
package indicator

// 导入 "github.com/markcheno/go-talib" 用于技术分析功能，特别是 ATR（平均真实范围）的计算。
import "github.com/markcheno/go-talib"

// SuperTrend 函数接收高价，低价，收盘价的浮点数切片，ATR期数，以及乘数因子。
// 它返回超趋势指标的浮点数切片。
// TR = MAX(当天最高价-当天最低价,|当天最高价-昨天收盘价|，|当天最低价-昨天收盘价|) ，TR（真实范围）：测量当天价格波动值，值越大波动越大，值越小，波动越小。
// ATR = 所有天数TR的和 /天数 ，计算了一段时间（如20天）内的平均价格波动值，提供了对市场近期波动性的一个整体感觉，帮助我们判断市场是比较平静还是动荡。
// atrPeriod这个参数决定了多少个周期（如天、周或月）的数据将被用来计算一个ATR值。比如，常见的选择包括14天、20天等
// factor的值决可以自定义， 提醒交易，值越小 提醒越快 同时也可能增加假信号的风险，值越大，提醒越慢，同时降低假信号的风险
// 基本上带 = 如第一天中点（(最高价+最低价)/2） +（factor(乘数因子)*ATR(平均真实范围)） 设置市场价格的一个上限指标，当价格突破这个上限时，通常被视为市场趋势向上的信号。
// 基本下带 = 如第一天中点（(最高价+最低价)/2） -（factor(乘数因子)*ATR(平均真实范围)） 设置为市场价格的一个下限指标，价格跌破这个下限时，通常被视为市场趋势向下的信号。
// 最终上带 = 当前周期的基本上带小于前一周期的最终上带，或者前一周期的收盘价大于前一周期的最终上带 两个条件满足一个，就更新当前的基本上带为最终上带吗，都不满足保持原样 ,最终上带可以被看作是股票上涨的一个最终界限，超过这个界限，解释为股票将持续走高的一个信号，买入的好时机
// 最终下带 = 当前周期的基本下带大于前一周期的最终下带，前一周期的收盘价小于前一周期的最终下带，满足一个当前周期的基本下带会被更新为当前周期的最终下带。最终下带可以看作股票最终下跌的界限，跌过这个界限，可能会持续下降，就会提示卖出的好时机，

// 注意1 最终上带和最终下带：这些带是在基本带的基础上，通过考虑价格的历史行为和之前的趋势动态进行调整后得出的。
// 注意2 基本上带和基本下带：它们是根据当前的价格数据和ATR计算出的初步界限。这些界限代表了在没有进一步调整之前的趋势方向的指示。
func SuperTrend(high, low, close []float64, atrPeriod int, factor float64) []float64 {
	// talib.Atr 所以这个方法自动帮我们计算ATR，只要有最低价，最高价，收盘价，周期
	atr := talib.Atr(high, low, close, atrPeriod)

	// 初始化基本上下带，最终上下带，和超趋势指标的切片。如果你有30天TR的数据，然后计算ATR，这个ATR也算是每天TR的平均值，你会得到30个ATR值  所以长度可以是atr
	basicUpperBand := make([]float64, len(atr)) //基本上带
	basicLowerBand := make([]float64, len(atr)) //基本下带
	finalUpperBand := make([]float64, len(atr)) //最终上带
	finalLowerBand := make([]float64, len(atr)) //最终下带
	superTrend := make([]float64, len(atr))     //超级趋势

	// 从第二个数据点开始迭代，因为超趋势需要前一个点的数据。
	for i := 1; i < len(basicLowerBand); i++ {
		// 基本上带 = 如第一天中点（(最高价+最低价)/2） +（factor(乘数因子)*ATR(平均真实范围)）
		basicUpperBand[i] = (high[i]+low[i])/2.0 + atr[i]*factor
		// 基本下带 = 如第一天中点（(最高价+最低价)/2） -（factor(乘数因子)*ATR(平均真实范围)）
		basicLowerBand[i] = (high[i]+low[i])/2.0 - atr[i]*factor

		// 计算最终上带，如果基本上带低于前一个最终上带，或者前一个收盘价高于前一个最终上带。那么基本上带等于最终上带
		if basicUpperBand[i] < finalUpperBand[i-1] || close[i-1] > finalUpperBand[i-1] {
			finalUpperBand[i] = basicUpperBand[i]
		} else {
			//否则最终上带没有改变，依然是等于上一个最终上带
			finalUpperBand[i] = finalUpperBand[i-1]
		}

		// 计算最终下带，如果基本下带高于前一个最终下带，或者前一个收盘价低于前一个最终下带。那么基本下带等于最终下带
		if basicLowerBand[i] > finalLowerBand[i-1] || close[i-1] < finalLowerBand[i-1] {
			finalLowerBand[i] = basicLowerBand[i]
		} else {
			///否则最终下带依然等于前一个的最终下带
			finalLowerBand[i] = finalLowerBand[i-1]
		}

		// 确定超趋势指标的值。
		//如果上一期最终上带等于上一期超级趋势，意味着k线是上升的， 因为最终上带本身就意味着价格上升
		if finalUpperBand[i-1] == superTrend[i-1] {
			//如果当前收盘价 close[i] 大于当前周期的最终上带 finalUpperBand[i] 这表明价格已经突破了上升趋势的界限，显示了强劲的上升动力。
			if close[i] > finalUpperBand[i] {
				superTrend[i] = finalLowerBand[i]
			} else {
				//如果收盘价小于最终上带的话，也是上升趋势，但是有所减缓
				superTrend[i] = finalUpperBand[i]
			}
		} else { // 如果上一期最终下带等于上一期超级趋势，意味着k线是下降的，因为最终下带就意味下降趋势
			// 如果收盘价<最终下带 表名下降趋势非常强势，需要谨慎
			if close[i] < finalLowerBand[i] {
				superTrend[i] = finalUpperBand[i]
			} else {
				//如果小于的话，也是下降趋势，但是有所减缓
				superTrend[i] = finalLowerBand[i]
			}
		}
	}

	// 返回计算出的超趋势指标切片。
	return superTrend
}
