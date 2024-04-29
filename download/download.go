package download

import (
	"context"
	"encoding/csv"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/xhit/go-str2duration/v2"

	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

/*
Downloader 结构体定义了一个数据下载器，其中包含了一个交易所数据源的接口实例。
NewDownloader 函数用于创建一个新的数据下载器实例。
Parameters 结构体定义了数据下载的时间参数，包括开始时间和结束时间。
Option 是一个函数类型，用于修改 Parameters 结构体的字段，例如设置下载的时间间隔或指定下载的天数。
WithInterval 函数返回一个 Option 类型的函数，用于设置下载数据的时间间隔。
WithDays 函数返回一个 Option 类型的函数，用于设置下载数据的时间范围。
candlesCount 函数计算给定时间范围内K线数据的数量和持续时间。
Download 方法是核心逻辑，用于下载数据并保存到CSV文件中。
方法接收交易对、时间间隔、输出文件路径和一系列可选参数。
方法首先尝试创建输出文件，然后根据传入的参数计算数据的时间范围。
下载过程分批进行，每批次获取一定数量的K线数据，并写入到CSV文件中。
在下载过程中，使用进度条显示下载进度。
最后，检查是否有未下载完全的数据，并记录警告信息。
最终，刷新CSV写入器，关闭进度条，记录下载完成的日志，并返回可能出现的错误信息。
这个下载器的设计灵活性很高，用户可以通过命令行参数动态配置下载的时间范围和输出文件路径，同时支持不同时间间隔的K线数据下载。
*/
// 意思就是如果interval 持续时间是1个小时batchSize = 500就是500个小时的数据点，如果是持续时间是1天，batchSize = 500就是500天的数据点，每个数据点就是一个k线图数据
const batchSize = 500

// Downloader 结构体，这个是获取数据源的结构体，获取的数据源包含着k线的数据时间戳，开盘价，收盘价（Close），最高价（High），最低价（Low），交易量（Volume）
type Downloader struct {
	exchange service.Feeder // exchange 是一个实现了 service.Feeder 接口的对象，用于获取数据。
}

// NewDownloader 函数创建并返回一个新的 Downloader 实例。
// 这意味着通过传入不同的实现到NewDownloader函数，你可以创建一个能够从不同数据源（即不同交易所）获取数据的Downloader实例。
func NewDownloader(exchange service.Feeder) Downloader {
	return Downloader{
		exchange: exchange, // 设置 exchange 属性为传入的 service.Feeder 实例。
	}
}

// Parameters 结构体，定义了数据下载的时间参数。
type Parameters struct {
	Start time.Time // Start 表示下载数据的起始时间。
	End   time.Time // End 表示下载数据的结束时间。
}

// Option 是一个函数类型，用于修改 Parameters 实例。
type Option func(*Parameters)

// WithInterval 函数返回一个 Option 类型的函数，用于设置下载数据的时间间隔。
func WithInterval(start, end time.Time) Option {
	return func(parameters *Parameters) {
		parameters.Start = start // 设置参数的起始时间。
		parameters.End = end     // 设置参数的结束时间。
	}
}

// WithDays 函数生成一个 Option 类型的函数，该函数用于设置数据下载的时间范围。
// 参数 `days` 表示从当前时间往回数的天数。
func WithDays(days int) Option {
	return func(parameters *Parameters) {
		//AddDate(0, 0, -days)  表示在此刻的时间加上年月日， 这里面的年，月，参数是0，日的话是传过来的负数，这段代码的意思就是开始时间倒退-days天
		parameters.Start = time.Now().AddDate(0, 0, -days)
		// 设置参数的结束时间为当前时间。
		parameters.End = time.Now()
	}
}

// candlesCount 函数计算给定时间范围内，k线的数量，还有，k线持续的时间
// start 和 end 定义了时间范围，timeframe 定义了每个数据点覆盖的时间长度。
// timeframe：这是一个字符串参数，表示每个数据点的时间长度，例如 "1h" 表示一小时，"1d" 表示一天。
func candlesCount(start, end time.Time, timeframe string) (int, time.Duration, error) {
	// 计算总持续时间。计算开始时间到结束时间的差值多少，就是开始到结束，持续了多少时间
	totalDuration := end.Sub(start)
	// str2duration.ParseDuration(timeframe) 来解析像如timeframe是 "1h30m" 这样的字符串时，函数会试图将这个字符串转换成一个 time.Duration 对象。 "1h30m" 代表90分钟，1秒 = 1,000,000,000纳秒因此如果转换成功，它将是 5,400,000,000,000 纳秒
	//转换为 time.Duration 对象使得时间相关的操作变得更加精确和灵活。这种转换允许开发者执行精确的时间计算，如加减时间点，设定定时器和延迟，以及管理事件的持续时间和间隔。使用 time.Duration 提高了代码的可读性和可维护性，减少了手动时间单位计算的复杂性和出错率，非常适合处理需要时间控制的多种编程场景。
	interval, err := str2duration.ParseDuration(timeframe)
	if err != nil {
		// 如果解析出错，返回错误。
		return 0, 0, err
	}
	// 如果 totalDuration 是24小时，而 interval 是1小时，那么 totalDuration / interval 的结果将是24，意味着在这24小时内可以完整包含24个1小时的K线。totalDuration / interval这个时间内包含多少k线图
	//interval一个k线的间隔时间
	return int(totalDuration / interval), interval, nil
}

// Download 方法，下载交易对的K线数据到CSV文件。
// output string这个字符串参数指定了下载数据要保存到的文件路径。这里的数据将被保存为CSV格式，可以被用于后续的数据分析或作为历史数据记录。
// 这是一个可变参数，允许传入多个Option类型的函数。每个Option函数可以修改Parameters结构的字段，比如调整数据下载的起始和结束时间。这提供了高度的灵活性，使调用者可以在调用方法时动态配置数据下载的具体参数。
func (d Downloader) Download(ctx context.Context, pair, timeframe string, output string, options ...Option) error {
	// 尝试创建输出文件，用于保存下载的数据。
	//根据output文件路径，创建一个文件，如果创建失败比如因为文件系统权限不足、磁盘空间不足、路径错误或其他系统问题就返回错误
	//os.Create 是 Go 语言标准库中的一个函数，属于 os 包。它用于在文件系统中创建一个新文件。如果指定的文件已经存在，os.Create 会将其长度截断为 0（即清空文件内容），确保返回的文件句柄是针对一个空文件。
	recordFile, err := os.Create(output)
	if err != nil {
		// 文件创建失败，返回错误。
		return err
	}

	// 获取当前时间。
	now := time.Now()
	// 设置默认的下载参数（起始时间为一月前，结束时间为当前时间）。
	parameters := &Parameters{
		Start: now.AddDate(0, -1, 0),
		End:   now,
	}

	// 应用所有传入的选项来修改下载参数。
	for _, option := range options {
		option(parameters)
	}

	// 校正开始时间到最近的整日开始（UTC时区）设置开始时间为标准时间， time.Date函数从原始的 parameters.Start 中提取出年份、月份和日期，时，分，秒，纳秒，设置为零，意思就是开始时间从午夜0点，即一天的开始时刻
	parameters.Start = time.Date(parameters.Start.Year(), parameters.Start.Month(), parameters.Start.Day(),
		0, 0, 0, 0, time.UTC)

	// 使用 now.Sub(parameters.End) 检查当前时间 now 与设定的结束时间 parameters.End 之间的差异。
	//预设结束时间在当前时间之前，那表明没有错误，然后调整为程序会将结束时间调整到该天的午夜0点，并且使用UTC时区。程序还是会将parameters.End调整到那一天的午夜0点，并使用UTC时区。这个调整的目的是标准化结束时间点，确保它始终在一天的开始：确保所有数据的时间戳都统一在一天的开始，便于数据处理和对比
	if now.Sub(parameters.End) > 0 {
		//这表示设定的结束时间 parameters.End 在当前时间 now 之前。因此，程序会将结束时间调整到该天的午夜0点，并且使用UTC时区。
		parameters.End = time.Date(parameters.End.Year(), parameters.End.Month(), parameters.End.Day(),
			0, 0, 0, 0, time.UTC)
	} else {
		//当 now.Sub(parameters.End) <= 0 时，表示预设的结束时间等于当前时间或者超过了当前时间，所以程序会把结束时间设置为当前时间。这样的逻辑确保了数据的有效性和及时性。
		parameters.End = now
	}

	// 计算需要下载的K线数量和每个K线的时间间隔。
	candlesCount, interval, err := candlesCount(parameters.Start, parameters.End, timeframe)
	if err != nil {
		// 时间间隔解析失败，返回错误。
		return err
	}
	//增加一个额外的K线：candlesCount++ 操作在得到的K线数量基础上增加1。这样做通常是为了确保覆盖完整的数据范围，特别是在时间范围的边界上，可能存在一些数据点未完全包括在内。
	candlesCount++
	// 记录日志：开始下载数据。记录下载多少根k线还有k线的持续时间，交易对
	log.Infof("Downloading %d candles of %s for %s", candlesCount, timeframe, pair)
	// 获取交易对的资产信息。
	info := d.exchange.AssetsInfo(pair)
	//csv.NewWriter 函数创建了一个新的 CSV 文件写入器。这个写入器与先前创建的文件 recordFile 绑定。 这个写入器 writer 将被用来向 CSV 文件写入一系列的字符串数组。每个字符串数组代表 CSV 文件的一行，通常包括各种数据字段，如在金融数据分析中的时间、开盘价、收盘价、最低价、最高价和交易量等。
	writer := csv.NewWriter(recordFile)

	//  创建了一个进度条实例，这个进度条的最大值被设置为 candlesCount，即预计要下载的K线的总数。这个进度条用于可视化表示数据下载的进展情况。
	/*
			func main() {
			bar := progressbar.Default(100) // 假设任务总量为100

			for i := 0; i < 100; i++ {
				bar.Add(1) // 每完成1%的任务，进度条更新1
				time.Sleep(10 * time.Millisecond) // 模拟任务执行时间
			}

			fmt.Println("任务完成！")
		}
	*/
	progressBar := progressbar.Default(int64(candlesCount))
	// 这行代码初始化了一个变量 lostData 来记录在数据下载过程中可能发生的k线丢失事件的数量。k线丢失可能由于多种原因，如网络问题、数据源问题等。
	lostData := 0
	//意思就是从2023年1月1日起，计划每七天下载一次数据，每次循环一个星期，第一次循环，isLastLoop = false，因为满七天 ，第二次也是isLastLoop = false满七天，最后一次开始日期：1月29日，预计结束日期：2月4日因为已经超过了这个月的下载数据，所以把isLastLoop 调为 true，调整结束日期为1月31日。因为isLastLoop 调为 true 就可以不用循环下载一个星期，到最后一个下载时间点就结束
	isLastLoop := false // 标记是否是最后一次循环。true 就是最后一次循环

	// 写入CSV文件的表头。就是CSV文件第一行的标题
	err = writer.Write([]string{
		"time", "open", "close", "low", "high", "volume",
	})
	if err != nil {
		// 写入失败，返回错误。
		return err
	}

	// 循环从 parameters.Start 开始，这是指定的开始时间，要 begin（当前循环的起始时间）仍然在 parameters.End（结束时间）之前。就继续循环，在每次循环迭代中，begin 时间会通过加上 interval * batchSize 来更新。这意味着每次循环结束时，begin 都会向前推进由 interval 和 batchSize 定义的总时间跨度。 interval 设为一小时batchSize = 500就是500小时的数据点，每次循环开始时间，就往后推500小时，直到，开始时间在结束时间之后
	for begin := parameters.Start; begin.Before(parameters.End); begin = begin.Add(interval * batchSize) {
		//这里计算的是每次循环周期的结束时间= 开始时间 + （间隔时间 x 数据点）
		end := begin.Add(interval * batchSize)
		// 这是每个周期的结束时间，一定要在总的结束时间之前防止超出整个任务设定的时间范围。
		if end.Before(parameters.End) {
			//如果一个小时的周期从10:00开始，按照原始的计划它将在11:00结束。然而11:00 也是下个周期的开始时间，防止重叠，我们会从结束时间减去一秒，使得周期实际上在10:59:59结束
			end = end.Add(-1 * time.Second)
		} else {
			// 如果计算出的周期结束时间不早于整个任务的结束时间，意味着这是最后一个处理周期
			end = parameters.End
			// 标记 isLastLoop 为 true，表示这是最后一次循环，之后不再继续
			isLastLoop = true
		}

		// 从数据源获取K线数据。
		candles, err := d.exchange.CandlesByPeriod(ctx, pair, timeframe, begin, end)
		if err != nil {
			// 数据获取失败，返回错误。
			return err
		}

		// 将K线数据写入CSV文件。
		//for 循环来遍历 candles 切片，其中每个 candle 代表一个时间段的交易数据，包括开盘价、最高价、最低价、收盘价和成交量等信息。
		//它的作用是将 candle 对象的数据转换成一个字符串切片（slice）。这个切片包含了K线数据的所有重要元素，格式化为字符串，便于存储和处理。info.QuotePrecision：这是一个参数，通常用于指定在转换数据时应该保留的小数位数。 意思就是将k线图数组数据格式化为字符串，里面的及格精度是我们设定的报价精度对吗比如精度是0.01，意思就是里面的价格也是保留两位小数
		for _, candle := range candles {
			err := writer.Write(candle.ToSlice(info.QuotePrecision))
			if err != nil {
				// 数据写入失败，返回错误。
				return err
			}
		}

		// 更新处理的K线计数和进度条。
		// countCandles 用于存储当前批次中获取的K线数据点的数量。candles 是一个包含多个K线数据的切片
		countCandles := len(candles)
		//  如果不是最后一个循环周期，k线丢失的数量 = 原先的数量 + 数据点-k线图数据的长度
		// 假设每批次你期望处理 500条数据（batchSize = 500），但在一个非最后的循环周期中，你只成功获取了 490 条数据（countCandles = 490） 丢失的数据 = 原来丢失的数据  + 期望的数据点 - 获取的数据点 。 假设原来丢失的数据是5 ，那么现在的丢失的数据 = 5 + 500-490 = 15 ，丢失的数据有可能是网络问题。
		if !isLastLoop {
			lostData += batchSize - countCandles
		}
		//进度条的主要作用是实时展示数据下载或处理的进度。每次循环中获取到的K线数据的数量（countCandles）会被添加到进度条的当前值中，这样用户就能看到进度条逐渐填满，直到任务完成。
		if err = progressBar.Add(countCandles); err != nil {
			// 更新进度条失败，记录警告。
			log.Warnf("update progresbar fail: %s", err.Error())
		}
	}

	// 关闭进度条。
	//循环结束之后已经将所有获取到的k线数据添加到数据条之后关闭进度条，,关闭进度条是在所有数据已经成功处理并且进度条已经显示为100%（或完全填满）之后的操作
	if err = progressBar.Close(); err != nil {
		// 关闭进度条
		log.Warnf("close progresbar fail: %s", err.Error())
	}

	// 如果存在未下载完全的数据（即预期的K线数量与实际下载的K线数量不符），记录警告。
	//如果有没有下载完的数据，就将一个警告消息添加到日志中
	if lostData > 0 {
		log.Warnf("%d missing candles", lostData)
	}

	// 刷新CSV写入器，确保所有数据都已写入文件。
	writer.Flush()
	// 记录日志，表示下载任务完成。
	log.Info("Done!")
	// 检查CSV写入过程中是否有错误发生，并返回这个错误。
	return writer.Error()
}
