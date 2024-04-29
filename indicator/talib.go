package indicator

// 引入github.com/markcheno/go-talib包，这是TA-Lib（技术分析库）的Go语言版本，提供了各种金融市场技术分析工具。
import "github.com/markcheno/go-talib"

// 定义MaType类型，等同于talib库中的MaType类型。TA-Lib 是一款广泛使用的技术分析软件库，能够用于金融市场的各种数据分析，比如股票、期货、外汇等市场的价格和交易量数据分析。它提供了包括移动平均线、相对强弱指数（RSI）、布林带等在内的超过 150 种不同的技术指标和数学函数。。
type MaType = talib.MaType

// 定义了多种移动平均线类型的常量，这些常量实际上是对talib包中定义的相应常量的引用。
// 例如，TypeSMA对应简单移动平均线，TypeEMA对应指数移动平均线等。
const (
	// 简单移动平均线 ，第一天的SMA等于收盘价，第四天的SMA 用1、2、3、4 天的收盘价之和除以4 ，如果要拿到第五天的SMA就是用前5天的收盘价除以5 ，我用这个方法算出1~8天的SMA 如果值一直是上升的，那么市场趋势可能上升，向下，市场可能趋势向下
	TypeSMA = talib.SMA
	//权重 = 2 /周期+1  周期如7天 权重等于 2/7+1= 1/4
	// 指数移动平均线 EMA =今天的收盘价×权重 + 昨天的EMA - 昨天的EMA x 权重，第一天因为没有前一天的值，EMA等于收盘价
	//就是如果短期的EMA(如10天) 穿过了长期的EMA(如50天)这种情况通常被称为“金叉”，它被视为一个买入信号或是上升趋势的开始 ， 如果短期EMA从上方穿过长期EMA向下，这种情况被称为“死叉”，通常被视为卖出信号或是下降趋势的开始
	TypeEMA = talib.EMA
	//这里的权重一般是一个递增取值，如给出5天的收盘价，那权重分别是12345，都是递增一一对应
	//WMA = (第一天权重 x 第一天收盘价)+(第二天权重 x 第二天收盘价) +(第三天权重 x 第三天收盘价)）/权重之和
	//算出WMA与给出一段时间的收盘价对比，值接近序列的最高点，可能表示一个上升趋势。值接近序列的最低点，可能表示一个下降趋势。如果WMA值位于最高点和最低点之间，特别是当价格波动较大时，它可能表示市场处于一种不确定状态或者反映了某种程度的波动性。
	TypeWMA = talib.WMA // 加权移动平均线
	//第一次EMA是用如4天的收盘价计算的，然后第二次EMA 是用4天的每天EMA计算的公式都一样，只是第一次用收盘价数据，第二次用每天的EMA数据 计算
	//DEMA = 2 x 第一次EMA - EMA的EMA
	//要有足够的数据天数收盘价算出EMA 和EMA的EMA ，如(10天，20天，50天) 这样推算 当短期DEMA（例如10日DEMA）上穿长期DEMA（例如50日DEMA），这被视为买入信号，表明趋势可能从下跌转向上涨。死叉：当短期DEMA下穿长期DEMA，这被视为卖出信号，表明趋势可能从上涨转向下跌。单线持续上涨，表示价格有可能上涨可以买入
	TypeDEMA = talib.DEMA // 双指数移动平均线
	//第一次EMA是用如4天的收盘价计算的 ， 二重EMA用的是第一次的EMA数据计算的，三重EMA用的是二重EMA的数据计算的
	//TEMA = 3 x 第一次EMA - 3 x 二重EMA +三重EMA
	//当短期TEMA（例如10日DEMA）上穿长期TEMA（例如50日DEMA），这被视为买入信号，表明趋势可能从下跌转向上涨。死叉：当短期TEMA下穿长期TEMA，这被视为卖出信号，表明趋势可能从上涨转向下跌。单线持续上涨，表示价格有可能上涨可以买入
	TypeTEMA = talib.TEMA // 三重指数移动平均线
	//TRIMA = N期SMA的总和/N
	//短期TRIMA 上行穿过长期TRIMA 称为“金叉”，这可能是一个买入信号，短期TRIMA 下行穿过长期TRIMA 称为“死叉”，这可能是一个卖出信号，当TRIMA线向上移动时，表示价格处于上升趋势；当TRIMA线向下移动时，表示价格处于下降趋势。
	TypeTRIMA = talib.TRIMA // 三角移动平均线
	//较小的fast_sc值会增加KAMA指标的灵敏度，使其更快地跟踪价格变化 slow_sc: 较大的slow_sc值确实会增加KAMA指标的平滑性，降低噪音的影响。这意味着KAMA指标的变化会更加缓慢，更平滑地反映价格趋势的变化，从而减少了虚假信号的出现。 N=5表示我们考虑最近的5个价格数据来计算ER
	//价格变动之和 = （第二个价格 - 第一个价格 ）+（第三价格-第二价格）以此类推
	//价格波动之和 =|（第二个价格 - 第一个价格 ）|+|（第三价格-第二价格）| 以此类推
	//效率比ER = 价格变动之和/价格波动之和
	//计算平滑系数（SC）= (计算效率比率（ER）x  (fast_sc - slow_sc) + slow_sc)^2
	//第二个KAMA   = 第一个KAMA +平滑系数SC x (第二个收盘价-第一个KAMA )
	//第一个KAMA 等于第一个收盘价
	//当短期KAMA线从下方向上穿过长期KAMA线时，我们可能会看到一个金叉信号，这可能是一个买入的时机，暗示着价格可能开始上升。相反，当短期KAMA线从上方向下穿过长期KAMA线时，我们可能会看到一个死叉信号，这可能是一个卖出的时机，暗示着价格可能开始下降。 单个上升暗示价格上升
	TypeKAMA = talib.KAMA // Kaufman自适应移动平均线
	//太复杂了使用go-talib 计算
	/* 	// 示例数据：收盘价
	   	closePrices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}

	   	// 计算MAMA和FAMA
	   	// 注意：fastLimit 和 slowLimit 参数可能需要根据你的需求调整
	   	fastLimit := 0.5
	   	slowLimit := 0.05
	   	mama, fama := talib.Mama(closePrices, fastLimit, slowLimit)
	*/
	//当MAMA线上穿FAMA线时，通常被视为买入信号，表示趋势可能正在向上变化。当MAMA线下穿FAMA线时，通常被视为卖出信号，表示趋势可能正在向下变化。
	TypeMAMA = talib.MAMA // MESA自适应移动平均线，通常与FAMA一起使用
	/* 	//太复杂了用go-talib 计算
	   	// 模拟一组收盘价数据
	   	prices := []float64{ 101.5, 102.3, 103.7, ...  }
	   	// 使用go-talib计算T3MA
	   	t3 := talib.T3(prices, 5, 0.7)
	   	fmt.Println("T3MA:", t3) */

	//当短期T3MA从下方穿越长期T3MA向上时，买入信号，可能表明趋势转向上升，反之亦然。如果单线上升表示，趋势上升，反之亦然
	TypeT3MA = talib.T3MA // 三重指数平滑移动平均线
)

// BB函数计算Bollinger Bands（布林带），这是一个常用于衡量价格高低和波动性的指标。
// 输入参数是价格数组(如两天的收盘价[20,30])、期间长度(如20天，20小时)、偏离度(常用为2)和移动平均线类型(如SMA，EMA)，返回上中下三条布林带的数值数组。
// deviation偏离度小于2 布林带变得更窄，这意味着价格更频繁地触及或超出布林带的边界，小的价格波动，也会跟进，导致获取的无用价格信息很多，偏离度等于2提供了一个平衡当价格触及或超出布林带的边界时，通常被视为比较强烈的市场动态信号。偏离度大于2会使布林带变宽，小的价格不波动不会触及布林带的边界，只有较大的价格波动才能触及布林带的边界，表示强烈的价格波动，市场的不稳定性
// 上带通常被视为市场的潜在阻力水平。当价格接近或突破上带时，市场可能被视为过度买入，这可能是价格反转或至少进行一定程度回调的信号。
// 中带代表了市场的“平均”或基准价格水平，常被用来判断市场趋势的方向。价格位于中带之上通常表明上升趋势，而位于中带之下则表明下降趋势。
// 下带通常被视为市场的潜在支撑水平。当价格接近或突破下带时，市场可能被视为过度卖出，这可能是价格反弹向上或至少稳定的信号。
func BB(input []float64, period int, deviation float64, maType MaType) ([]float64, []float64, []float64) {
	// 返回上中下三条布林带的数值数组。
	return talib.BBands(input, period, deviation, deviation, maType)
}

// DEMA函数计算双指数移动平均线（DEMA），这是一种更快、更平滑且对市场价格变动更敏感的移动平均线。
// 输入参数是价格数组和期间长度，返回DEMA的数值数组。
// 其中每个DEMA元素反映了对应时间点上基于过去'period'天（包括当天）的DEMA双指数移动平均值。 意思就是说我算周期20天的  算出第20的DEMA 表示这个是基于过期20的DEMA 平均值计算的
func DEMA(input []float64, period int) []float64 {
	// 返回DEMA的数值数组里面对应period每个之间点的DEMA数据
	return talib.Dema(input, period)
}

// EMA函数计算指数移动平均线（Exponential Moving Average），这是一种流行的趋势追踪工具，
// 它比简单移动平均线（SMA）对价格的最近变动给予更多权重，从而能更快地反应价格变动。
// 输入参数是价格数组和期间长度，返回EMA的数值数组。
// 其中每个元素反映了对应时间点上基于过去'period'天（包括当天）的指数移动平均值。
func EMA(input []float64, period int) []float64 {
	// 使用talib库的Ema方法根据提供的价格数据和周期长度计算并返回EMA数值数组。
	// 这个数组包含了从输入数据数组中第'period'天开始的每个时间点对应的EMA计算值。
	return talib.Ema(input, period)
}

// HTTrendline 像是一个出头鸟，很敏锐的感受到价格的变动，到价格上升的时候，但HTTrendline开始出现向下的弯曲，这可能意味着趋势即将改变转跌，当价格下跌之后，HTTrendline开始出现向上的弯曲，表示即将转升
// 接收参数是一个收盘价数组，这个组需要足够长的时间序列来捕捉市场的周期性和趋势变化
// 返回包含了计算出的希尔伯特变换趋势线的值， 每个元素对应于输入价格数组中的一个时间点如天数的趋势线值。
func HTTrendline(input []float64) []float64 {
	// 调用talib库的HtTrendline方法，根据提供的价格数据计算并返回希尔伯特变换趋势线的值。
	return talib.HtTrendline(input)
}

// KAMA函数计算考夫曼自适应移动平均线（Kaufman Adaptive Moving Average），
// 它是一种能够自适应市场波动性变化的移动平均线。相比于传统的移动平均线，
// 输入参数 这是一个浮点数数组，通常代表了一系列的价格数据（如股票的收盘价 也可以是其他）。这是KAMA计算的基础数据，period 指定用于计算KAMA的时间周期。周期长度决定了,平均线计算时考虑的数据点数量
// 返回值这是一个浮点数数组，包含了计算出的KAMA值。每个元素对应于输入价格数组中的一个时间点的KAMA值。x
func KAMA(input []float64, period int) []float64 {
	// 调用talib库的Kama方法，根据提供的价格数据和周期长度计算并返回KAMA数值数组。
	// 这个数组包含了从输入数据数组中每个时间点对应的KAMA计算值。
	return talib.Kama(input, period)
}

// MA函数使用TA-Lib库来计算不同类型的移动平均线（MA）。移动平均线是通过平滑过去价格数据来揭示价格趋势的一种技术分析工具。根据选定的移动平均线类型，MA可以是简单的（SMA）、指数的（EMA）、加权的（WMA）等等。
// input 通常代表了一系列的价格数据（比如股票的收盘价）。这是MA计算的基础数据。 period 这是一个整数，指定用于计算MA的时间周期。周期长度决定了均线计算时考虑的数据点数量，maType MaType这是一个枚举或特定类型的参数，指定要计算的移动平均线的类型。maType MaType
// []float64: 这是一个浮点数数组，包含了计算出的MA值。每个元素对应于输入价格数组中的一个时间点的MA值。
// 短期MA上穿长期MA：被称为“黄金交叉”，意味着短期趋势向上，可能预示着价格即将上涨，反之亦然
func MA(input []float64, period int, maType MaType) []float64 {
	// 调用talib库的Ma方法，根据提供的价格数据、周期长度和移动平均线类型
	// 计算并返回MA数值数组。
	return talib.Ma(input, period, maType)
}

// MAMA - moving average convergence/divergence
// 这个函数使用输入的价格数据（例如股票或加密货币的价格），以及两个速率限制参数，
// 来计算移动平均收敛/发散（MAMA）和它的伴随指标FAMA（Following Adaptive Moving Average）。
// 这两个指标一起帮助分析价格趋势的方向和强度，以及可能的趋势反转。
// 使用talib库的Mama函数进行计算，这个库提供了高效且精确的技术分析算法实现。
// slowLimit float64: 慢速限制参数，减少MAMA指标的敏感度。值越大，指标对价格变化的反应越慢。
// 返回两个参数 第一个切片是MAMA值的序列，第二个切片是FAMA值的序列
func MAMA(input []float64, fastLimit float64, slowLimit float64) ([]float64, []float64) {
	return talib.Mama(input, fastLimit, slowLimit)
}

// input []float64: 输入数据数组，通常为时间序列数据，如股票的日收盘价。
// periods []float64: 一个浮点数数组，表示希望计算移动平均值的不同周期。例如，可以同时考虑10天、20天、50天的移动平均。
// minPeriod int: 考虑计算的最小周期。这个参数用于过滤掉太短的周期，帮助避免过度的市场噪声。
// maxPeriod int: 考虑计算的最大周期。这个参数限制了分析的最长周期，帮助专注于更加相关的时间范围。
// maType MaType: 移动平均类型的枚举值，决定了移动平均的计算方式。不同的计算方法可以包括简单移动平均（SMA）、指数移动平均（EMA）等。
// 使用talib库的MaVp函数进行计算，该库是金融市场分析中常用的技术分析工具集之一，提供了一系列的函数来计算各种交易指标。
// 短期上升穿过长期 就是看涨信号，反之亦然
func MaVp(input []float64, periods []float64, minPeriod int, maxPeriod int, maType MaType) []float64 {
	return talib.MaVp(input, periods, minPeriod, maxPeriod, maType)
}

// 计算一组k线的收盘价的中点值，如周期是3，看这三天的最高点和最低点，之和除以2，如果周期是4，就取4组数据
// 如果MidPoint持续上升，可能表明市场在该周期内呈现上涨趋势；相反，如果MidPoint持续下降，则可能表示市场处于下跌趋势。
func MidPoint(input []float64, period int) []float64 {
	return talib.MidPoint(input, period)
}

// 计算一段时间内所有k线的中点值，全部k线的最低价与最高价之和除以2 ，如周期为3天，就利用这三天所有K线中的最高价最高价与最低价之和除以2
// 取一段时间里的k线的中点值，作为一个分界线，作为参考，如果接下来的价格高于这根线，有可能上涨，如果低于，有可能下降
func MidPrice(high []float64, low []float64, period int) []float64 {
	return talib.MidPrice(high, low, period)
}

// SAR (Parabolic Stop and Reverse) 是一种用于识别市场趋势和潜在反转点的技术分析指标。
// 它通过在价格图表上绘制一系列点来工作，这些点表示潜在的停止和反转水平。
// high []float64: 一系列最高价数据，通常为一定时间内每个交易周期（如每日、每小时）的最高交易价格。
// low []float64: 一系列最低价数据，对应于相同时间周期内的最低交易价格。
// inAcceleration float64: 加速因子，用于控制SAR点接近价格的速度。加速因子的初始值通常设置在0.02左右。
// inMaximum float64: 加速因子的最大值，用于限制加速因子的增加，确保SAR指标的灵敏度保持在合理的范围内。通常设置为0.2。
// 如果你设置inMaximum为0.2，无论市场价格如何变动，inAcceleration（加速因子）增加到0.2后就不会再继续增加了
// inAcceleration 就像一个点随着k线移动只要到达最高值或最低值，就逐渐接近价格 设置越大接近越快，越低接近越慢，逐步调整到，最低或最高价格
// Parabolic SAR点通常以一系列的小点或圆点表示，上升趋势：当SAR点连续出现在K线图的下方时，这通常被解读为上升趋势，是持有或买入的信号。下降趋势：相反，当SAR点连续出现在K线图的上方时，这通常被解读为下降趋势，是卖出或空头的信号
func SAR(high []float64, low []float64, inAcceleration float64, inMaximum float64) []float64 {
	return talib.Sar(high, low, inAcceleration, inMaximum)
}

// SARExt（扩展的Parabolic SAR）提供了交易者更多的灵活性来手动设置SAR指标的参数，以适应不同的市场条件和交易策略。下降趋势：SARExt点连续出现在价格图表的上方，这通常被解读为下降趋势，可能是一个卖出信号或避免买入的建议。
func SARExt(high []float64, low []float64,
	startValue float64, // SAR的初始值，算法开始计算时SAR点的起始位置
	offsetOnReverse float64, // 当趋势反转时，SAR值的偏移量
	accelerationInitLong float64, // 上升趋势中SAR点的初始加速因子
	accelerationLong float64, // 上升趋势中，每次新高时加速因子的增加量
	accelerationMaxLong float64, // 上升趋势中加速因子的最大值
	accelerationInitShort float64, // 下降趋势中SAR点的初始加速因子
	accelerationShort float64, // 下降趋势中，每次新低时加速因子的增加量
	accelerationMaxShort float64, // 下降趋势中加速因子的最大值
) []float64 {
	return talib.SarExt(high, low, startValue, offsetOnReverse, accelerationInitLong, accelerationLong,
		accelerationMaxLong, accelerationInitShort, accelerationShort, accelerationMaxShort) // 调用talib库的SarExt函数，根据提供的参数计算扩展的Parabolic SAR值
}

// SMA 计算给定数据序列的简单移动平均值。
// input []float64: 价格数据序列，通常是某个金融资产的收盘价，但也可以是开盘价、最高价或最低价，按时间顺序排列。
// period int: 移动平均的周期长度。这决定了平均计算中包含多少个数据点。例如，10天的SMA会计算最近10天的平均价格。
// 收集该周期内每一天的价格数据。这通常是每天的收盘价或其他 然后相加之和除以价格数量(天数)
//当k线位于SMA线的上方并且SMA线向上倾斜时，表示市场处于上升趋势。这表明价格趋向于上升，可能是一个买入信号。反之亦然

func SMA(input []float64, period int) []float64 {
	return talib.Sma(input, period)
}

// // T3 计算给定数据序列的三重指数移动平均值（Triple Exponential Moving Average，T3）。
// T3是一种基于三重指数平滑技术的移动平均线，它可以更快速地适应价格变动，同时保持足够的平滑性。
// // inVFactor float64: T3指标的平滑系数。平滑系数的值通常在0.1到1之间，影响平均值的平滑程度。较高的值会导致更平滑的线条，但可能会延迟趋势的反转。
// 就相当于三重EMA 短期上升穿过长期，买入信号，可能价格会上涨，反之亦然
func T3(input []float64, period int, inVFactor float64) []float64 {
	return talib.T3(input, period, inVFactor)
}

// 短期TEMA上行穿过长期TEMA称为“金叉”，这可能是一个买入信号，短期TRIMA 下行穿过长期TRIMA 称为“死叉”，这可能是一个卖出信号，当TRIMA线向上移动时，表示价格处于上升趋势；当TRIMA线向下移动时，表示价格处于下降趋势。
func TEMA(input []float64, period int) []float64 {
	return talib.Tema(input, period)
}

// 当短期TRIMA线从下方向上穿过长期TRIMA线时，我们可能会看到一个金叉信号，这可能是一个买入的时机，暗示着价格可能开始上升,反之亦然
func TRIMA(input []float64, period int) []float64 {
	return talib.Trima(input, period)
}

// 当短期 WMA线从下方向上穿过长期 WMA线时，我们可能会看到一个金叉信号，这可能是一个买入的时机，暗示着价格可能开始上升,反之亦然
func WMA(input []float64, period int) []float64 {
	return talib.Wma(input, period)
}

// ADX 计算给定数据序列的平均方向运动指数（Average Directional Index，ADX）。
// ADX是一种用于衡量市场趋势强度的技术指标。它基于价格波动的方向性变化来确定趋势的强度，而不是趋势的方向。
// ADX值越高，趋势越强。小于20：趋势较弱，可能是震荡市或者趋势即将形成的阶段。20到40：趋势较为明显，市场处于趋势运动的阶段。大于40：趋势非常强烈，市场可能已经过热，可能出现回调或者转势的可能性较大。当ADX线向上移动时，表示市场趋势的强度正在增加，趋势可能会持续。相反，当ADX线向下移动时，表示市场趋势的强度正在减弱，市场可能进入震荡或趋势转变的阶段
func ADX(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Adx(high, low, close, period)
}

// ADXR值表示趋势的强度，它是 ADX（Average Directional Index）的平均值
// 小于20：趋势较弱，可能是震荡市或者趋势即将形成的阶段。小于20：趋势较弱，可能是震荡市或者趋势即将形成的阶段。小于20：趋势较弱，可能是震荡市或者趋势即将形成的阶段。与ADX相似，当ADXR线向上移动时，表示市场趋势的强度正在增加，趋势可能会持续。相反，当ADXR线向下移动时，表示市场趋势的强度正在减弱，市场可能进入震荡或趋势转变的阶段。
func ADXR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.AdxR(high, low, close, period)
}

// input []float64: 价格数据序列，通常是某个金融资产的收盘价，按时间顺序排列。
// fastPeriod int: 快速移动平均线的周期长度。这决定了快速移动平均线的计算所使用的数据量。
// slowPeriod int: 慢速移动平均线的周期长度。这决定了慢速移动平均线的计算所使用的数据量。
// maType MaType: 移动平均线类型，包括简单移动平均（SMA）、指数移动平均（EMA）等。用于指定计算移动平均线时所采用的算法。
// 如果快速周期为3，如第三天的快速移动平均线Fast MA = 前三天的收盘价平均值，慢速周期为5，如慢速周期Slow MA如第五天 =  前五天的平均值， 快速周期，周期的天数要少，慢速周期，周期比较长
// 当APO线从下方向上穿过零线时，表示快速移动平均线超过了慢速移动平均线，可能预示着价格的上升趋势开始形成，这可能是买入信号。相反，当APO线从上方向下穿过零线时，可能表示价格的下降趋势正在形成，这可能是卖出信号。当快速移动平均线从下方向上穿过慢速移动平均线时，APO值变为正值，表示价格的上升趋势可能正在加速，这可能是一个买入信号。反之，当快速移动平均线从上方向下穿过慢速移动平均线时，APO值变为负值，表示价格的下降趋势可能正在加速，这可能是一个卖出信号。
func APO(input []float64, fastPeriod int, slowPeriod int, maType MaType) []float64 {
	return talib.Apo(input, fastPeriod, slowPeriod, maType)
}

// Aroon 指标是一种技术分析指标，用于衡量价格趋势的变化和趋势的强度。它由两条线组成：Aroon 上升线和 Aroon 下降线。
// Aroon 上升线衡量了在给定时间周期内最高价距离最近的最高价的位置，以及最高价距离时间周期的比例。
// Aroon 下降线衡量了在给定时间周期内最低价距离最近的最低价的位置，以及最低价距离时间周期的比例。
// 返回 Aroon 上升线和 Aroon 下降线的数组。
// 当Aroon上升线从下方向上穿过Aroon下降线时，通常被解读为价格即将进入上升趋势，可能是一个买入信号。 反之亦然关注Aroon值的变化趋势，如果Aroon值持续增加，可能表示价格趋势的持续性较强，可以考虑继续跟随趋势 反之亦然
func Aroon(high []float64, low []float64, period int) ([]float64, []float64) {
	return talib.Aroon(high, low, period)
}

// AroonOsc 计算 Aroon 振荡指标。
// Aroon 振荡指标是 Aroon 上升线与 Aroon 下降线之间的差异。
// AroonOsc 是上升线与下降线之间的距离,通常用他们之差求出来
// 当 Aroon Oscillator 的数值为正时，表示 Aroon 上升线在 Aroon 下降线之上，这可能暗示着价格处于上升趋势；反之，当 Aroon Oscillator 的数值为负时，表示 Aroon 上升线在 Aroon 下降线之下，这可能暗示着价格处于下降趋势。
func AroonOsc(high []float64, low []float64, period int) []float64 {
	return talib.AroonOsc(high, low, period)
}

// 资金动向指标（Balance Of Power）衡量买方和卖方力量之间的平衡。它的计算基于当日收盘价与当日最高价和最低价的关系。
// 参数 inOpen 是开盘价数组，high 是最高价数组，low 是最低价数组，close 是收盘价数组。
// 当 BOP 的数值为正时，表示买方力量较强，这意味着收盘价接近当日的最高价，这通常暗示着市场买盘较为活跃，可能预示着价格上涨的趋势。当 BOP 的数值为负时，表示卖方力量较强，这意味着收盘价接近当日的最低价，这通常暗示着市场卖盘较为活跃，可能预示着价格下跌的趋势。
func BOP(inOpen []float64, high []float64, low []float64, close []float64) []float64 {
	return talib.Bop(inOpen, high, low, close)
}

// Chande 动量振荡器（CMO）是一种衡量价格变动速度的指标。它基于最近一段时间内的价格变动量，可以帮助确定价格的超买和超卖情况。
// 参数 input 是价格数据数组，period 是指定的时间周期。
// 返回 Chande 动量振荡器的数组。
// 当 CMO 的数值为正时，表示价格的上涨动能较强，市场可能处于超买状态。这可能暗示着价格可能会出现回调或下跌的趋势。当 CMO 的数值为负时，表示价格的下跌动能较强，市场可能处于超卖状态。这可能暗示着价格可能会出现反弹或上涨的趋势。
func CMO(input []float64, period int) []float64 {
	return talib.Cmo(input, period)
}

// 商品通道指数（CCI）是一种衡量价格相对于其统计平均值的差异的指标，用于评估价格的波动性和可能的超买或超卖情况。
// 参数 high 是最高价数组，low 是最低价数组，close 是收盘价数组，period 是指定的时间周期。
// 返回商品通道指数的数组。
// 超买和超卖区域：一般来说，CCI指标大于+100被视为超买信号，表示价格可能已经过高；而CCI指标小于-100被视为超卖信号，表示价格可能已经过低。这些阈值可以根据具体市场条件和交易策略进行调整。
func CCI(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Cci(high, low, close, period)
}

// 方向性运动指数（DX）是一种衡量市场趋势强度的指标，它基于股价的高、低和收盘价的变化。
// 参数 high 是最高价数组，low 是最低价数组，close 是收盘价数组，period 是指定的时间周期。
// 返回方向性运动指数的数组。
// DX指标的数值可以用来衡量市场的趋势强度。一般来说，较高的DX数值表示趋势的强度较高，而较低的DX数值表示趋势的强度较低。 短期DX上升穿过长期DX线，看涨，反之亦然
func DX(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Dx(high, low, close, period)
}

// // MACD 计算移动平均收敛与发散（Moving Average Convergence/Divergence）指标。
// 参数 input 是收盘价数组，fastPeriod 是快速移动平均线的周期长度，slowPeriod 是慢速移动平均线的周期长度，signalPeriod 是信号线的周期长度。
// 返回 MACD 指标的三个数组：MACD 线、信号线、和 MACD 柱状图。
// signalPeriod自己定义：短的信号线周期会导致更频繁的信号，因为它会更快地反应价格的变化。这可能会增加交易信号的数量，但也可能包含更多的噪音，增加假消息。较长的周期会减少信号的数量，减少假消息，但可能会错过一些价格变化。它会忽略掉一些短期的价格波动，给出更稳定的信号
func MACD(input []float64, fastPeriod int, slowPeriod int, signalPeriod int) ([]float64, []float64, []float64) {
	return talib.Macd(input, fastPeriod, slowPeriod, signalPeriod)
}

// MACDExt 计算移动平均收敛与发散（MACD）指标的扩展版本，允许用户指定不同的移动平均类型。
// 参数 input 是收盘价数组，fastPeriod 是快速移动平均线的周期长度，fastMAType 是快速移动平均线的类型，
// slowPeriod 是慢速移动平均线的周期长度，inSlowMAType 是慢速移动平均线的类型，
// signalPeriod 是信号线的周期长度，signalMAType 是信号线的类型。
// 返回 MACD 指标的三个数组：MACD 线、信号线、和 MACD 柱状图。
// 当MACD线从下方向上穿过信号线时，这被视为买入信号，表示市场可能出现上涨趋势。反之亦然，当MACD柱状图处于正值时，表示MACD线高于信号线，这可能暗示着上涨趋势的加强；相反，当MACD柱状图处于负值时，表示MACD线低于信号线，这可能暗示着下跌趋势的加强，当MACD线向上移动并保持正值时，表示市场处于上涨趋势；而当MACD线向下移动并保持负值时，表示市场处于下跌趋势。
// signalPeriod自己定义：短的信号线周期会导致更频繁的信号，因为它会更快地反应价格的变化。这可能会增加交易信号的数量，但也可能包含更多的噪音，增加假消息。较长的周期会减少信号的数量，减少假消息，但可能会错过一些价格变化。它会忽略掉一些短期的价格波动，给出更稳定的信号
func MACDExt(input []float64, fastPeriod int, fastMAType MaType, slowPeriod int, inSlowMAType MaType,
	signalPeriod int, signalMAType MaType) ([]float64, []float64, []float64) {
	return talib.MacdExt(input, fastPeriod, fastMAType, slowPeriod, inSlowMAType, signalPeriod, signalMAType)
}

// MACDFix 计算固定周期的移动平均收敛与发散（MACD）指标。
// 参数 input 是收盘价数组，signalPeriod 是信号线的周期长度。
// 返回 MACD 指标的三个数组：MACD 线、信号线、和 MACD 柱状图。
// MACD 柱状图表示 MACD 线与信号线之间的差异。当 MACD 线从下方向上穿过信号线时，通常被解释为买入信号，表示市场可能出现上涨趋势。相反，当 MACD 线从上方向下穿过信号线时，通常被解释为卖出信号，表示市场可能出现下跌趋势。另外，MACD 柱状图处于正值时，表示 MACD 线高于信号线，可能暗示着上涨趋势的加强；
func MACDFix(input []float64, signalPeriod int) ([]float64, []float64, []float64) {
	return talib.MacdFix(input, signalPeriod)
}

// 返回一个[]float64类型的数组，每个元素对应一个周期的- DI值，反映了该周期内价格下跌的强度。
// 它属于方向移动系统（Directional Movement System）的一部分，主要衡量价格下跌趋势的力度。
// MinusDI 函数用于计算给定数据集的负方向指示器（-DI），这是一个用于分析市场趋势强度和方向的技术指标。
// MinusDI值增加：表示下跌趋势增强，卖方力量增强。
// MinusDI值减小：表示下跌趋势力度减弱，但不直接表示上升趋势的增强。
// PlusDI高于MinusDI，这被视为上升趋势的标志，因为它表明买方力量超过卖方力量。如果MinusDI高于PlusDI，则被视为下降趋势的标志，表明卖方力量超过买方力量。  两条线比较，然后比的是一个整体，谁在上方谁在下方， PlusDI在上方被视为上升趋势的标志，在下方被视为下降趋势的标志
func MinusDI(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.MinusDI(high, low, close, period)
}

// MinusDM 函数计算给定周期内的负方向移动（MinusDM）值。负方向移动是一个衡量资产价格下跌动力的指标，
// 它通过比较连续两个周期（如两天）的最低价，来识别市场的下跌趋势。
// 返回 []float64: 一个浮点数数组，每个元素代表相应周期的MinusDM值。这个数组的长度与输入数组的长度相同。
// 如果 MinusDM 值在一定周期内持续增加，这通常表明市场下跌动力在增强，卖方市场控制力增强。如果 MinusDM 值在减少，这可能表示市场下跌动力减弱，卖方控制力正在下降。
// 当PlusDM值高于MinusDM时，这表明市场的上升趋势强于下降趋势，可能是买入信号。 MinusDM值高于PlusDM时，下降趋势可能占优势，可能是卖出信号。
func MinusDM(high []float64, low []float64, period int) []float64 {
	return talib.MinusDM(high, low, period)
}

// MFI 函数计算给定周期内的资金流量指标（MFI），这是一个结合价格和成交量信息来衡量买卖压力的动量指标。
// MFI 被用来识别市场的过度买入或过度卖出条件，可以帮助交易者做出更加明智的交易决策。
//   - volume []float64: 代表观察期内每个周期的成交量数组。每个元素对应一个周期的成交量，是衡量买卖力度的重要指标。
//   - period int: 指定计算MFI时考虑的时间周期数量，决定了计算中包括的数据范围。
//   - []float64: 一个浮点数数组，每个元素代表相应周期的MFI值。MFI值范围从0到100，用于评估市场的买卖条件。
//     一般而言，MFI高于80表示市场可能过度买入，低于20表示市场可能过度卖出。
func MFI(high []float64, low []float64, close []float64, volume []float64, period int) []float64 {
	return talib.Mfi(high, low, close, volume, period)
}

// Momentum 函数计算给定周期内的动量指标值。动量是衡量资产价格变化速度的技术分析工具，
// 它通过比较当前价格与过去某一特定周期前的价格来计算得出。
// []float64: 一个浮点数数组，每个元素代表相应周期的动量值。动量值可以是正的也可以是负的
// 正值意味上涨，负值意味下跌
func Momentum(input []float64, period int) []float64 {
	return talib.Mom(input, period)
}

// PlusDI 函数计算给定周期内的正方向指示器（+DI），这是一个用于分析市场上升趋势强度的技术指标。
// +DI 是方向移动系统（Directional Movement System）的一部分，主要通过比较连续两个周期内的最高价格来评估上升动力。
// []float64: 一个浮点数数组，每个元素代表相应周期的+DI值。+DI值用于评估市场的上升趋势强度，
// PlusDI高于MinusDI，这被视为上升趋势的标志，因为它表明买方力量超过卖方力量。如果MinusDI高于PlusDI，则被视为下降趋势的标志，表明卖方力量超过买方力量。  两条线比较，然后比的是一个整体，谁在上方谁在下方， PlusDI在上方被视为上升趋势的标志，在下方被视为下降趋势的标志
// PlusDI值上升它表明买方力量在增加，即市场的上升动力在加强。它表明买方力量在增加，即市场的上升动力在加强。
func PlusDI(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.PlusDI(high, low, close, period)
}

// PlusDM 函数计算给定周期内的正方向移动（PlusDM）值。正方向移动是一个衡量资产价格上升动力的指标，
// 它通过比较连续两个周期（如两天）的最高价，来识别市场的上升趋势。
// []float64: 一个浮点数数组，每个元素代表相应周期的PlusDM值。这个值可以用来分析市场的上升趋势强度，
// PlusDM值增加表示市场上升动力在增强，即当前周期的最高价与前一周期的最高价之间差异增大，显示买方控制力增强。减少的PlusDM值减少的PlusDM值
// 当PlusDM值高于MinusDM时，这表明市场的上升趋势强于下降趋势，可能是买入信号。 MinusDM值高于PlusDM时，下降趋势可能占优势，可能是卖出信号。
func PlusDM(high []float64, low []float64, period int) []float64 {
	return talib.PlusDM(high, low, period)
}

// PPO函数计算百分比价格振荡器（Percentage Price Oscillator），是一种动量指标，用于衡量两个移动平均线（一个快速和一个慢速）之间的比例差异。
// PPO指标显示的是两个移动平均线的差值除以慢速移动平均线的值，结果以百分比形式表示。这有助于识别价格趋势的强度和方向。
// - input []float64: 表示输入价格数组，通常是收盘价，但也可以是其他价格数据（如开盘价、最高价、最低价等）。
// - fastPeriod int: 定义快速移动平均线的周期数。较小的数值会使指标对价格变化更敏感。
// - slowPeriod int: 定义慢速移动平均线的周期数。较大的数值提供了对趋势的更平滑且较慢的响应。
// - maType MaType: 指定用于计算移动平均线的类型（如简单移动平均SMA，指数移动平均EMA等）。
// []float64: 一个浮点数数组，每个元素代表相应周期的PPO值。PPO值可以帮助分析市场趋势的动量和方向。
// PPO线上升：表明短期动量在加强，短期价格上涨速度超过长期价格上涨速度，可能是买入信号。PPO线下降：表明短期动量在减弱，短期价格下跌速度超过长期价格下跌速度，可能是卖出信号。
// PPO线穿越信号线向上：当PPO线从下方穿越其信号线（PPO的移动平均线）向上时，这可能被解读为买入信号。
// PPO线穿越信号线向下：当PPO线从上方穿越其信号线向下时，这可能被解读为卖出信号。
// 信号线就是EMA
func PPO(input []float64, fastPeriod int, slowPeriod int, maType MaType) []float64 {
	return talib.Ppo(input, fastPeriod, slowPeriod, maType)
}

// ROCP函数计算给定周期内价格变化率的百分比，作为衡量动量的指标。
// []float64: 一个浮点数数组，每个元素代表对应时间点的ROCP值。数组的长度与输入数组相同，数组的前'period'个值可能不会有有效的ROCP值，具体取决于实现的处理方式。
// 当ROCP值由负转正时，表明价格开始上升，可能考虑买入。 当ROCP值由正转负时，表明价格开始下降，可能考虑卖出。
func ROCP(input []float64, period int) []float64 {
	return talib.Rocp(input, period)
}

// ROC函数计算给定周期内的价格变化率，这是一个动量指标，用于衡量资产价格与之前特定周期价格的相对变化。
// []float64: 一个浮点数数组，每个元素代表对应时间点的ROC值。数组的前几个值（具体数量取决于周期长度）可能不会有有效的ROC值， 因为它们没有足够的历史数据来形成完整的周期。这意味着实际有效的ROC值从数组的第'period'+1个元素开始计算。
// ROC指标可以用来分析市场趋势的强度和潜在的转折点。一个增加的ROC值可能表明市场动量在加强，而一个减少的ROC值可能表明市场动量在减弱。
// 交易者可以根据ROC值的变化来调整其买卖策略，例如，当ROC值由负转正时可能考虑买入，而当ROC值由正转负时可能考虑卖出。
func ROC(input []float64, period int) []float64 {
	return talib.Roc(input, period)
}

// ROCR函数计算给定周期内的价格变化比率，这是一个动量指标，用于衡量资产价格相对于之前特定周期价格的相对变化。
// []float64: 一个浮点数数组，每个元素代表对应时间点的价格变化比率。数组的前几个值（具体数量取决于周期长度）可能不会有有效的计算结果，因为它们没有足够的历史数据来形成完整的周期。这意味着实际有效的价格变化比率值从数组的第'period'+1个元素开始计算。
// 当 ROCR 值从负值转变为正值时，可能是买入信号，表示价格开始上升。当 ROCR 值从正值转变为负值时，可能是卖出信号，表示价格开始下降。正值：表示当前价格高于过去特定周期的价格，即价格上涨。正值：表示当前价格高于过去特定周期的价格，即价格上涨。
func ROCR(input []float64, period int) []float64 {
	return talib.Rocr(input, period)
}

// ROCR100函数计算给定周期内的价格变化比率，以百分比形式表示。它衡量了资产价格相对于之前特定周期价格的相对变化，并将结果乘以100以获得百分比表示。
// []float64: 一个浮点数数组，每个元素代表对应时间点的价格变化比率，以百分比形式表示。数组的前几个值（具体数量取决于周期长度）可能不会有有效的计算结果，因为它们没有足够的历史数据来形成完整的周期。这意味着实际有效的价格变化比率值从数组的第'period'+1个元素开始计算。
// 当 ROCR100 的值增加时，表示当前价格变化的幅度增大，可能意味着市场动量正在增强。当 ROCR100 的值减少时，表示当前价格变化的幅度减小，可能意味着市场动量正在减弱。，当 ROCR100 值大幅增加时，可能是买入信号；当 ROCR100 值大幅减少时，可能是卖出信号。
func ROCR100(input []float64, period int) []float64 {
	return talib.Rocr100(input, period)
}

// RSI函数用于计算相对强弱指数（RSI），这是一种常用的技术指标，用于衡量价格变动的速度和幅度，进而评估资产的超买和超卖情况。
//   - []float64: 一个浮点数数组，每个元素代表对应时间点的相对强弱指数（RSI）值。数组的长度与输入数组的长度相同。
//
// RSI 值通常在 0 到 100 之间，数值越高表示超买情况越严重，数值越低表示超卖情况越严重。
// 当RSI值高于70时，通常被视为超买信号，表明市场可能过度买入，价格可能会出现调整或下跌。当RSI值低于30时，通常被视为超卖信号，表明市场可能过度卖出，价格可能会出现反弹或上涨。
func RSI(input []float64, period int) []float64 {
	return talib.Rsi(input, period)
}

// Stoch函数用于计算慢速随机指标（Slow Stochastic Indicator），它是一种常见的技术指标，用于衡量资产价格的动量和超买/超卖情况。
// fastKPeriod (int): 这是计算随机振荡指标中的 %K 线所需的时间周期长度。这个值定义了用于计算高、低和收盘价格的滑动窗口大小。
// slowKPeriod (int): 一旦计算出快速 %K 值，这个参数定义了用于平滑快速 %K 值的移动平均的时间周期。这个平滑过程生成了慢速 %K 值。
// - slowKMAType MaType: 用于计算慢速K线的移动平均类型，慢速k值值利用快速k值已经计算好的部分从中取一部分平均。
// - slowDPeriod int: 慢速%D值就利用已经算好的慢速K值的值平均一下
// - slowDMAType MaType: 用于计算慢速%D线的移动平均类型，可以是简单移动平均、指数移动平均等
// - []float64: 一个浮点数数组，代表每个周期的慢速随机指标的%K值。
// - []float64: 一个浮点数数组，代表每个周期的慢速随机指标的%D值。
// 函数Stoch的返回值包括两个浮点数数组，%K值和%D值
// 超买/超卖信号：当%K线和%D线位于高位（如80以上），可能表示市场处于超买状态；当它们位于低位（如20以下），可能表示市场处于超卖状态。
// 金叉：当%K线从下向上穿过%D线，可以视为买入信号，表示可能的上升趋势开始。
// 死叉：当%K线从上向下穿过%D线，可以视为卖出信号，表示可能的下跌趋势开始。
// 如周5天数据，快速K线的值 =(当前收盘价 - 5天最低价)/(5天最高价-5天最低价) x 100
func Stoch(high []float64, low []float64, close []float64, fastKPeriod int, slowKPeriod int,
	slowKMAType MaType, slowDPeriod int, slowDMAType MaType) ([]float64, []float64) {

	return talib.Stoch(high, low, close, fastKPeriod, slowKPeriod, slowKMAType, slowDPeriod, slowDMAType)
}

// StochF的函数，它计算了快速随机指标（Fast Stochastic Oscillator），这是一个常用于技术分析的动量指标，主要用于评估证券价格的闭市情况相对于其价格范围的位置，以预测价格走势。
// fastDMAType MaType中的MaType是 各种线的类型如 SMA, ,EMA , WMA
// []float64, []float64)：函数返回两个浮点数切片。第一个切片是快速随机值（%K值），第二个切片是其平滑移动平均（%D值，即快速随机指标的信号线）
// 查找交叉：%K线和%D线的交叉点可以指示潜在的买入或卖出信号。%K线上穿%D线通常被视为买入信号，%K线下穿%D线被视为卖出信号。
// 超买超卖：%K线和%D线的值通常在0到100之间。值接近100可能表示资产超买，而值接近0可能表示资产超卖。
// 如周5天数据，快速K线的值 =(当前收盘价 - 5天最低价)/(5天最高价-5天最低价) x 100
// 快速D值就是快速k值得平均数
func StochF(high []float64, low []float64, close []float64, fastKPeriod int, fastDPeriod int,
	fastDMAType MaType) ([]float64, []float64) {

	return talib.StochF(high, low, close, fastKPeriod, fastDPeriod, fastDMAType)
}

// StochRSI（随机相对强弱指数）指标的读取和解释主要基于其在0到1（或者0%到100%）范围内的值，以及%K线与%D线之间的相互作用。
// 超买条件：当StochRSI值接近1（或100%）时，表明资产可能处于超买状态。这意味着价格可能过高，且存在回调或下跌的风险。
// 超卖条件：相反，当StochRSI值接近0（或0%）时，表明资产可能处于超卖状态。这意味着价格可能过低，且存在反弹或上升的机会。
// 金叉：当%K线（较快线）从下方穿越%D线（较慢线或其移动平均线）向上时，被视为买入信号。这表明动量可能正在转向正面，价格可能即将上涨。
// 死叉：当%K线从上方穿越%D线向下时，被视为卖出信号。这表明动量可能正在减弱，价格可能即将下跌。
func StochRSI(input []float64, period int, fastKPeriod int, fastDPeriod int, fastDMAType MaType) ([]float64,
	[]float64) {

	return talib.StochRsi(input, period, fastKPeriod, fastDPeriod, fastDMAType)
}

// Trix函数计算TRIX（三重指数平滑移动平均）指标，Trix函数计算TRIX（三重指数平滑移动平均）指标，这是一个旨在识别和确认金融市场数据中的趋势，以及信号潜在的反转或趋势变化的动量振荡器。
// []float64: 表示TRIX指标值的浮点数数组。[]float64: 表示TRIX指标值的浮点数数组。[]float64: 表示TRIX指标值的浮点数数组。
// 向上穿越零线：当TRIX线从下方穿越零线向上时，这通常被视为市场进入看涨趋势的信号，可能是买入的时机。向下穿越零线：当TRIX线从上方穿越零线向下时，这通常被视为市场进入看跌趋势的信号，可能是卖出的时机。
// TRIX为正表示当前趋势向上，市场动量增强，可能意味着继续持有或买入。为负表示当前趋势向下，市场动量减弱，可能意味着考虑退出或卖出。
func Trix(input []float64, period int) []float64 {
	return talib.Trix(input, period)
}

// UltOsc函数计算终极振荡器（Ultimate Oscillator, UltOsc）指标，这是一种综合利用不同时间周期的价格数据来测量市场动量的技术分析工具。通过结合短期、中期和长期的周期，终极振荡器旨在减少单一周期振荡器可能遇到的假信号问题，提供更准确的市场趋势信号。
// - period1 int: 短期周期的长度，用于计算终极振荡器的第一部分。
// - period2 int: 中期周期的长度，用于计算终极振荡器的第二部分。
// - period3 int: 长期周期的长度，用于计算终极振荡器的第三部分
// 如计算7天、14天和28天周期
// 超买区域：当UltOsc的值高于某个阈值（如70）时，市场可能处于超买状态，表明价格可能会回调或下跌。
// 超卖区域：当UltOsc的值低于另一个阈值（如30）时，市场可能处于超卖状态，表明价格可能会反弹或上涨 无论选择哪个阈值，都应该通过历史数据进行回测自定义的
func UltOsc(high []float64, low []float64, close []float64, period1 int, period2 int, period3 int) []float64 {
	return talib.UltOsc(high, low, close, period1, period2, period3)
}

// WilliamsR 函数计算威廉姆斯百分比范围（Williams %R）指标，这是一种动量指标，用于识别超买和超卖水平。
// 它测量当前收盘价相对于过去一段时间内最高价和最低价的位置。%R值通常在-100到0的范围内，其中接近-100表示超卖水平，
// 接近0表示超买水平。该指标有助于预测价格趋势的反转点。
// []float64: 代表Williams %R指标值的浮点数数组。这些值可以用来分析市场的超买或超卖状态，并辅助交易者做出买卖决策。
func WilliamsR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.WillR(high, low, close, period)
}

// Ad 函数计算累积/派发线（Accumulation/Distribution Line, A/D）指标， 这是一种用于衡量资金流入和流出的量化工具。A/D指标结合了价格和成交量，旨在显示资金是流入市场还是流出市场。
// 通过比较收盘价与最高和最低价的范围，以及考虑相应的成交量，A/D线能够提供价格走势背后的资金流动情况的线索。
// []float64: 代表累积/派发线指标值的浮点数数组。A/D线的上升趋势表明买方控制市场，而下降趋势则表明卖方占优势。该指标可以用来确认趋势或预警可能的趋势变化。
// 当累积/派发线呈上升趋势时，表示资金正在流入市场，通常预示着价格可能会上涨。
// 当累积/派发线呈下降趋势时，表示资金正在流出市场，通常预示着价格可能会下跌。
func Ad(high []float64, low []float64, close []float64, volume []float64) []float64 {
	return talib.Ad(high, low, close, volume)
}

// AdOsc 函数计算累积/派发震荡器（Accumulation/Distribution Oscillator，AdOsc）指标， 该指标是根据累积/派发线（Accumulation/Distribution Line）的变化率计算得出的。
//   - volume []float64: 一个浮点数数组，代表每个周期的成交量。
//   - fastPeriod int: 快速周期的长度，用于计算累积/派发震荡器的快速线。
//   - slowPeriod int: 慢速周期的长度，用于计算累积/派发震荡器的慢速线。
//
// []float64: 代表累积/派发震荡器指标值的浮点数数组。该指标通常是以两条线的形式呈现，[]float64: 代表累积/派发震荡器指标值的浮点数数组。该指标通常是以两条线的形式呈现，
// 当快速线从下方穿过慢速线时，可能暗示着资金流速正在加快，市场可能会出现买入压力，价格可能会上涨。
// 当快速线从上方穿过慢速线时，可能暗示着资金流速正在减缓，市场可能会出现卖出压力，价格可能会下跌。
func AdOsc(high []float64, low []float64, close []float64, volume []float64, fastPeriod int,
	slowPeriod int) []float64 {
	return talib.AdOsc(high, low, close, volume, fastPeriod, slowPeriod)
}

// OBV 函数计算平衡成交量指标（On Balance Volume，OBV），这是一种量价指标，用于衡量资金流入和流出的情况。
// []float64: 代表每个周期的平衡成交量指标值的浮点数数组。OBV指标的正负值表示资金流入和流出的方向， 当OBV上升时表示资金流入市场，可能预示着价格上涨；当OBV下降时表示资金流出市场，可能预示着价格下跌。
// 当OBV上升时，表示资金正在流入市场，可能预示着价格上涨。当OBV上升时，表示资金正在流入市场，可能预示着价格上涨。
func OBV(input []float64, volume []float64) []float64 {
	return talib.Obv(input, volume)
}

// ATR 函数计算真实波幅指标（Average True Range，ATR），这是一种用于衡量价格波动性的技术指标。
// []float64: 代表每个周期的真实波幅指标值的浮点数数组。真实波幅指标反映了市场波动的实际程度， 较大的ATR值表示市场波动较大，较小的ATR值表示市场波动较小。
func ATR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Atr(high, low, close, period)
}

// NATR 函数计算归一化真实波幅指标（Normalized Average True Range，NATR），这是真实波幅指标（ATR）的一种归一化版本。
// []float64: 代表每个周期的归一化真实波幅指标值的浮点数数组。NATR指标与价格水平无关，而更专注于波动性的测量。
// 当NATR的数值增加时，表示市场的波动性增加，价格可能更加波动。当NATR的数值减小时，表示市场的波动性减少，价格可能更加稳定。
func NATR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Natr(high, low, close, period)
}

// TRANGE 函数计算真实波幅（True Range，TRANGE）指标，它是用于衡量价格波动性的技术指标之一。
// []float64: 代表每个周期的真实波幅指标值的浮点数数组。真实波幅指标反映了市场在一段时间内的价格波动幅度， 它通常用于计算平均真实波幅指标（ATR）等其他技术指标。
// 当TRANGE的数值较大时，表示市场的波动性较高，价格变动幅度较大；当TRANGE的数值较小时，表示市场的波动性较低，价格相对稳定。
func TRANGE(high []float64, low []float64, close []float64) []float64 {
	return talib.TRange(high, low, close)
}

// AvgPrice 函数计算平均价格指标（Average Price，AvgPrice），这是一种用于衡量一定时间内的平均价格水平的技术指标。
// []float64: 代表每个周期的平均价格指标值的浮点数数组。平均价格指标可以帮助分析市场价格的整体水平，并结合其他技术指标一起使用，进行更全面的市场分析和交易决策。
// 当AvgPrice的数值上升时，表示价格整体上升，市场可能处于上涨趋势；当AvgPrice的数值下降时，表示价格整体下降，市场可能处于下跌趋势。
func AvgPrice(inOpen []float64, high []float64, low []float64, close []float64) []float64 {
	return talib.AvgPrice(inOpen, high, low, close)
}

// MedPrice 函数计算中位价格指标（Median Price，MedPrice），这是一种衡量价格水平的技术指标。
// 当MedPrice的数值上升时，表示价格整体上升，市场可能处于上涨趋势；当MedPrice的数值下降时，表示价格整体下降，市场可能处于下跌趋势。
func MedPrice(high []float64, low []float64) []float64 {
	return talib.MedPrice(high, low)
}

// TypPrice 函数计算典型价格指标（Typical Price，TypPrice），这是一种衡量价格水平的技术指标。
// 典型价格是指在一定时间段内的三个价格（最高价、最低价和收盘价）的平均值。
// []float64: 代表每个周期的典型价格指标值的浮点数数组。典型价格指标可以帮助分析市场价格的整体水平，  并结合其他技术指标一起使用，进行更全面的市场分析和交易决策。
// 当TypPrice的数值上升时，表示价格整体上升，市场可能处于上涨趋势；当TypPrice的数值下降时，表示价格整体下降，市场可能处于下跌趋势。
func TypPrice(high []float64, low []float64, close []float64) []float64 {
	return talib.TypPrice(high, low, close)
}

// WCLPrice 函数计算加权收盘价指标（Weighted Close Price，WCLPrice），这是一种衡量价格水平的技术指标。
// []float64: 代表每个周期的加权收盘价指标值的浮点数数组。加权收盘价指标可以帮助分析市场价格的整体水平，并结合其他技术指标一起使用，进行更全面的市场分析和交易决策。
// 当WCLPrice的数值上升时，表示价格整体上升，市场可能处于上涨趋势；当WCLPrice的数值下降时，表示价格整体下降，市场可能处于下跌趋势。
func WCLPrice(high []float64, low []float64, close []float64) []float64 {
	return talib.WclPrice(high, low, close)
}

// HTDcPeriod 函数计算 Hilbert Transform - Dominant Cycle Period 指标（HTDcPeriod），这是 Hilbert 变换技术的一部分，用于识别市场价格数据中的主导周期。
// []float64: 代表每个周期的 HTDcPeriod 指标值的浮点数数组。HTDcPeriod 可以帮助分析市场的周期性特征，有助于识别主导周期并用于制定相应的交易策略。
// 当HTDcPeriod的数值增加时，表示主导周期更长；当HTDcPeriod的数值减小时，表示主导周期更短。周期指的是波动性的周期
func HTDcPeriod(input []float64) []float64 {
	return talib.HtDcPeriod(input)
}

// HTDcPhase 计算 Hilbert Transform - Dominant Cycle Phase，即希尔伯特变换 - 主导周期相位。
// 返回值: 一个浮点数数组，代表检测到的主导周期的相位值。
// DcPhase值上升可能表示主导周期的相位也在随着时间推移而增加。 周期可能是波动周期，或者上升，或下降的周期
func HTDcPhase(input []float64) []float64 {
	return talib.HtDcPhase(input)
}

// HTPhasor 函数计算 Hilbert Transform - Phasor Components（希尔伯特变换 - 相量分量）指标，它用于提取价格序列中的快速和缓慢成分，有助于识别价格趋势和周期性行为。
// HTPhasor返回两个浮点数数组，分别代表快速和缓慢成分的相量分量。
// 如果快速成分的相量分量在增加，而缓慢成分的相量分量在减少，这可能表明价格正在经历快速的短期波动，但整体上处于较缓慢的长期趋势中。
// 如果两者的变化方向相反，可能表明市场正处于变化的过渡阶段，可能会出现趋势反转或价格波动的加速。
func HTPhasor(input []float64) ([]float64, []float64) {
	return talib.HtPhasor(input)
}

// HTSine 函数用于计算 Hilbert Transform - SineWave Components（希尔伯特变换 - 正弦波分量）指标，它用于提取价格序列中的正弦波分量。
// HTSine 返回两个浮点数数组，分别表示正弦波分量的快速和缓慢周期。
// 分析快速周期和缓慢周期的振幅变化。振幅的增加可能表示价格波动的加剧，而振幅的减少可能表示价格波动的减弱。
func HTSine(input []float64) ([]float64, []float64) {
	return talib.HtSine(input)
}

// HTTrendMode 函数用于计算 Hilbert Transform - Trend Mode（希尔伯特变换 - 趋势模式）指标，该指标用于确定价格序列中的趋势模式。
// HTTrendMode 返回一个浮点数数组，表示趋势模式指标的数值。
// 当趋势模式指标上升时，可能表示价格走势具有向上的趋势性质；而当趋势模式指标下降时，则可能表示价格走势具有向下的趋势性质。
func HTTrendMode(input []float64) []float64 {
	return talib.HtTrendMode(input)
}

// Beta 函数用于计算贝塔（Beta）指标，该指标衡量一个证券相对于市场的价格变动情况。
// 贝塔指标可以帮助投资者了解某个证券相对于整个市场的价格波动情况，以及其对市场变动的敏感程度。
// 贝塔值为1：该证券（货币）的价格波动与市场完全一致。贝塔值大于1：该证券（货币）的价格波动比市场波动更剧烈。贝塔值小于1：该证券（货币）的价格波动比市场波动更平缓。
// 如以太坊（ETH）的贝塔值为1，那么它的价格波动与比特币（BTC）的价格波动会完全一致。这意味着无论比特币的价格是上涨还是下跌，以太坊的价格变动也会与之一致，但幅度可能略有不同。
func Beta(input0 []float64, input1 []float64, period int) []float64 {
	return talib.Beta(input0, input1, period)
}

// Correl 函数用于计算两个数据序列之间的相关性。相关性是衡量两个变量之间关系的一种统计量，它表示两个变量之间的线性关系强度和方向。
// 返回值 []float64: 一个浮点数数组，代表计算出的相关性值。值的范围在-1到1之间，其中1表示完全正相关，-1表示完全负相关，0表示无相关性。 就是两组收盘价或其他的数据
func Correl(input0 []float64, input1 []float64, period int) []float64 {
	return talib.Correl(input0, input1, period)
}

// LinearReg 函数计算给定数据序列的线性回归值。线性回归是一种用于估计变量之间线性关系的统计方法，它可以通过拟合一条直线来描述两个变量之间的关系。
// 返回值 []float64: 一个浮点数数组，代表给定数据序列的线性回归值。返回的数组长度与输入数据序列长度相同。
// 收盘价的线性回归值表示根据过去一段时间内的收盘价数据如果线性回归值持续上升，可能暗示着未来价格也会上涨，反之亦然。
func LinearReg(input []float64, period int) []float64 {
	return talib.LinearReg(input, period)
}

// LinearRegAngle 函数计算给定数据序列的线性回归角度。线性回归角度是线性回归线的斜率以角度表示的度量，它描述了数据序列的线性趋势变化速率。
// 返回值 []float64: 一个浮点数数组，代表给定数据序列的线性回归角度。返回的数组长度与输入数据序列长度相同。
// 。当线性回归角度为正时，表示数据序列呈现上升趋势；当线性回归角度为负时，表示数据序列呈现下降趋势；当线性回归角度接近于零时，表示数据序列趋势相对平稳。
func LinearRegAngle(input []float64, period int) []float64 {
	return talib.LinearRegAngle(input, period)
}

// 这个函数用于计算给定数据序列的线性回归截距值。
// 所谓线性回归截距是指线性回归模型中直线与 y 轴的交点的位置。在金融领域中，线性回归截距可以用来衡量某个指标的基准值或初始值。
func LinearRegIntercept(input []float64, period int) []float64 {
	return talib.LinearRegIntercept(input, period)
}

// 这个函数用于计算给定数据序列的线性回归斜率值。线性回归斜率表示了线性回归模型中直线的斜率，即自变量变化一个单位时，因变量相应变化的幅度
// 趋势方向： 斜率的正负表示了数据序列的趋势方向。当斜率为正时，表示趋势为上升；当斜率为负时，表示趋势为下降。
// 趋势强度： 斜率的绝对值越大，表示趋势的变化速度越快，趋势强度越大；而绝对值较小则表示趋势变化缓慢。
func LinearRegSlope(input []float64, period int) []float64 {
	return talib.LinearRegSlope(input, period)
}

// 这个函数用于计算给定数据序列的标准差值。标准差是一种衡量数据波动性或变异程度的统计量，它表示数据点相对于平均值的分散程度。
// inNbDev 则是标准差的倍数系数，用于确定标准差的范围。 较大的倍数系数会扩大标准差的范围，使得价格波动被认为更加波动性较大。相反，较小的倍数系数会缩小标准差的范围，使得价格波动被认为更加稳定。
// 使用StdDev计算出的标准差数值可以用来观察数据的波动情况。如果标准差值在一段时间内持续增大，说明价格波动性正在增强；反之，如果标准差值在缩小，说明价格波动性在减小。
func StdDev(input []float64, period int, inNbDev float64) []float64 {
	return talib.StdDev(input, period, inNbDev)
}

// TSF函数用于计算时间序列预测（Time Series Forecast），它是一种基于历史数据进行预测的技术分析工具。
// []float64: 一个浮点数数组，表示对应输入数据序列的时间序列预测值。
// 首先，需要准备一定长度的历史数据序列，这些数据可以是价格序列或其他时间序列数据，例如股票价格、销售量等。准备好的历史数据序列和预测周期作为参数传递给TSF函数，调用函数进行预测。根据TSF函数返回的预测结果如果预测值逐渐上升，则可能暗示着未来趋势将继续上涨；反之，如果预测值逐渐下降，则可能暗示着未来趋势将下跌
func TSF(input []float64, period int) []float64 {
	return talib.Tsf(input, period)
}

// Var（方差）函数用于计算给定周期内数据序列的方差。
// 返回一个浮点数数组，代表每个周期内数据序列的方差值。
// 方差是衡量数据离散程度的指标，数值越大表示数据波动越大，反之数值越小表示数据波动越小。
func Var(input []float64, period int) []float64 {
	return talib.Var(input, period)
}

/* 数学变换函数的集合，用于对数据进行数学运算和转换。 */

// Acos 函数计算输入数据序列中每个元素的反余弦值（arccosine）。
// 参数 input 是包含输入数据序列的浮点数数组。
// 返回值是一个浮点数数组，包含了输入序列中每个元素的反余弦值。
// 反余弦函数的定义域是[-1, 1]，值域是[0, π]。因此，Acos 函数的输出将在0到π之间。如果输入数据序列中的值超出了[-1, 1]的范围，则可能导致无效的输出。
// input 可以是任何你想要进行反余弦运算的数值序列比如价格、指标值、量等。Acos 函数将应用于这个输入序列中的每个元素，计算出对应的反余弦值，并返回一个包含这些值的新序列。
func Acos(input []float64) []float64 {
	return talib.Acos(input)
}

// Asin 函数用于计算输入序列中每个元素的反正弦值。
// 这在数学和统计分析中是一个常见的操作，用于处理角度或弧度值，并进行相关的数学运算。
func Asin(input []float64) []float64 {
	return talib.Asin(input)
}

// Atan(input []float64) []float64: 计算数组中每个元素的反正切值。用于处理角度和斜率，常用于将直角坐标系转换为极坐标系。
func Atan(input []float64) []float64 {
	return talib.Atan(input)
}

// Ceil(input []float64) []float64: 对数组中的每个元素向上取整到最近的整数。常用于数值分析中的取整操作。
func Ceil(input []float64) []float64 {
	return talib.Ceil(input)
}

// Cos(input []float64) []float64: 计算数组中每个元素的余弦值。适用于波形分析和信号处理中的周期性变化。
func Cos(input []float64) []float64 {
	return talib.Cos(input)
}

// Cosh(input []float64) []float64: 计算数组中每个元素的双曲余弦值。常用于特定的数学和物理问题。
func Cosh(input []float64) []float64 {
	return talib.Cosh(input)
}

// Exp(input []float64) []float64: 计算数组中每个元素的指数。用于指数增长或衰减模型，如人口增长模型。
func Exp(input []float64) []float64 {
	return talib.Exp(input)
}

// Floor(input []float64) []float64: 对数组中的每个元素向下取整到最近的整数。常用于数值分析中的取整操作。
func Floor(input []float64) []float64 {
	return talib.Floor(input)
}

// Ln(input []float64) []float64: 计算数组中每个元素的自然对数。适用于增长速率和时间衰减的分析。
func Ln(input []float64) []float64 {
	return talib.Ln(input)
}

// Log10(input []float64) []float64: 计算数组中每个元素的以10为底的对数。常用于处理对数尺度下的数据，如pH值、声级。
func Log10(input []float64) []float64 {
	return talib.Log10(input)
}

// Sin(input []float64) []float64: 计算数组中每个元素的正弦值。适用于波形分析和信号处理中的周期性变化。
func Sin(input []float64) []float64 {
	return talib.Sin(input)
}

// Sinh(input []float64) []float64: 计算数组中每个元素的双曲正弦值。用于特定的数学和物理问题。
func Sinh(input []float64) []float64 {
	return talib.Sinh(input)
}

// Sqrt(input []float64) []float64: 计算数组中每个元素的平方根。用于距离和能量等概念的计算。
func Sqrt(input []float64) []float64 {
	return talib.Sqrt(input)
}

// Tan(input []float64) []float64: 计算数组中每个元素的正切值。用于角度的转换和测量。
func Tan(input []float64) []float64 {
	return talib.Tan(input)
}

// Tanh(input []float64) []float64: 计算数组中每个元素的双曲正切值。用于特定的数学和物理问题。
func Tanh(input []float64) []float64 {
	return talib.Tanh(input)
}

/* 数学运算符函数 */

// Add(input0, input1 []float64) []float64: 将两个数组对应元素相加。用于数据的组合和整合分析。
func Add(input0, input1 []float64) []float64 {
	return talib.Add(input0, input1)
}

// Div(input0, input1 []float64) []float64: 将两个数组对应元素相除。用于计算比率或效率等指标。
func Div(input0, input1 []float64) []float64 {
	return talib.Div(input0, input1)
}

// Max(input []float64, period int) []float64: 计算数组中指定周期的最大值。用于金融分析中的高点
// 寻找或者科学研究中寻找数据范围。
func Max(input []float64, period int) []float64 {
	return talib.Max(input, period)
}

// MaxIndex(input []float64, period int) []float64: 找出数组中指定周期的最大值的索引。常用于确定极值点在时间序列中的位置。
func MaxIndex(input []float64, period int) []float64 {
	return talib.MaxIndex(input, period)
}

// Min(input []float64, period int) []float64: 计算数组中指定周期的最小值。用于金融分析中的低点寻找或科学研究中寻找数据范围。
func Min(input []float64, period int) []float64 {
	return talib.Min(input, period)
}

// MinIndex(input []float64, period int) []float64: 找出数组中指定周期的最小值的索引。常用于确定极值点在时间序列中的位置。
func MinIndex(input []float64, period int) []float64 {
	return talib.MinIndex(input, period)
}

// MinMax(input []float64, period int) ([]float64, []float64): 同时计算数组中指定周期的最小值和最大值。用于快速获取时间序列数据的范围。
func MinMax(input []float64, period int) ([]float64, []float64) {
	return talib.MinMax(input, period)
}

// MinMaxIndex(input []float64, period int) ([]float64, []float64): 同时找出数组中指定周期的最小值和最大值的索引。用于分析时间序列中极值点的位置。
func MinMaxIndex(input []float64, period int) ([]float64, []float64) {
	return talib.MinMaxIndex(input, period)
}

// Mult(input0, input1 []float64) []float64: 将两个数组对应元素相乘。用于数据分析中的比例和产品计算。
func Mult(input0, input1 []float64) []float64 {
	return talib.Mult(input0, input1)
}

// Sub(input0, input1 []float64) []float64: 将两个数组对应元素相减。用于计算差异或增长率。
func Sub(input0, input1 []float64) []float64 {
	return talib.Sub(input0, input1)
}

// Sum(input []float64, period int) []float64: 计算数组中指定周期的元素总和。用于移动平均或累计指标的计算。
func Sum(input []float64, period int) []float64 {
	return talib.Sum(input, period)
}
