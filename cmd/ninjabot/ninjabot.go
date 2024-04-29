package main

// 导入必要的包
import (
	"log" // 用于记录错误信息
	"os"  // 用于访问系统操作，如命令行参数

	// 导入ninjabot包，用于下载数据、交互交易所和其他服务
	"github.com/rodrigo-brito/ninjabot/download"
	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/service"

	"github.com/urfave/cli/v2" // 导入urfave/cli库，用于创建命令行界面
)

/*
导入必要的包，包括日志记录 (log) 和系统操作 (os)，以及用于下载数据、与交易所交互和其他服务的相关包。

使用 urfave/cli 库创建一个命令行应用程序，命名为 "ninjabot"，并定义其名称、帮助名称和描述信息。

定义了一个名为 "download" 的命令，用于下载历史数据。该命令包含了一系列选项，包括交易对、下载天数、起始日期、结束日期、时间帧、输出文件路径和是否下载期货数据等。

当用户执行 "download" 命令时，根据用户的输入选项选择相应的数据源（现货或期货市场），然后准备下载选项，包括下载天数和时间范围。如果用户未正确指定时间范围，则会记录错误并退出。

使用指定的选项执行数据下载操作，并将结果输出到指定的输出文件路径中。

最后，运行应用程序并处理命令行输入，如果发生错误，则记录错误并退出。下载的数据保存到用户指定的输出文件中。在命令行选项中，通过 --output 或 -o 参数指定输出文件的路径和文件名。

用户可以在命令行中指定要保存数据的文件路径和名称，例如 --output ./btc.csvs
*/
func main() {
	// 创建一个新的cli应用
	app := &cli.App{
		Name:     "ninjabot",                   // 应用程序的名称
		HelpName: "ninjabot",                   // 帮助文档中使用的名称
		Usage:    "Utilities for bot creation", // 应用程序的描述
		// 定义应用程序支持的命令列表
		Commands: []*cli.Command{
			{
				Name:     "download",                 // 命令的名称
				HelpName: "download",                 // 帮助文档中使用的命令名称
				Usage:    "Download historical data", // 命令的描述
				// 命令的选项
				Flags: []cli.Flag{
					// 定义一个字符串选项，用于指定交易对
					&cli.StringFlag{
						Name:     "pair",
						Aliases:  []string{"p"}, // 选项的短名称
						Usage:    "eg. BTCUSDT", // 选项的说明
						Required: true,          // 此选项为必须
					},
					// 定义一个整数选项，用于指定下载数据的天数
					&cli.IntFlag{
						Name:     "days",
						Aliases:  []string{"d"},
						Usage:    "eg. 100 (default 30 days)",
						Required: false, // 此选项非必须
					},
					// 定义一个时间戳选项，用于指定下载数据的起始日期
					&cli.TimestampFlag{
						Name:     "start",
						Aliases:  []string{"s"},
						Usage:    "eg. 2021-12-01",
						Layout:   "2006-01-02", // 时间的格式
						Required: false,        // 此选项非必必须
					},
					// 定义一个时间戳选项，用于指定下载数据的结束日期
					&cli.TimestampFlag{
						Name:     "end",
						Aliases:  []string{"e"},
						Usage:    "eg. 2020-12-31",
						Layout:   "2006-01-02",
						Required: false, // 此选项非必须
					},
					// 定义一个字符串选项，用于指定时间帧
					&cli.StringFlag{
						Name:     "timeframe",
						Aliases:  []string{"t"},
						Usage:    "eg. 1h",
						Required: true, // 此选项为必须
					},
					// 定义一个字符串选项，用于指定输出文件的路径
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "eg. ./btc.csv",
						Required: true, // 此选项为必须
					},
					// 定义一个布尔选项，用于指定是否下载期货数据
					&cli.BoolFlag{
						Name:     "futures",
						Aliases:  []string{"f"},
						Usage:    "true or false",
						Value:    false, // 默认值为false
						Required: false, // 此选项非必须
					},
				},
				// 当download命令被执行时调用的动作
				Action: func(c *cli.Context) error {
					var (
						exc service.Feeder // 定义
						err error
					)

					// 根据用户是否指定"futures"选项来选择相应的数据源
					if c.Bool("futures") {
						// 从币安期货市场获取数据
						exc, err = exchange.NewBinanceFuture(c.Context)
						if err != nil {
							return err // 如果有错误，返回错误
						}
					} else {
						// 从币安现货市场获取数据
						exc, err = exchange.NewBinance(c.Context)
						if err != nil {
							return err // 如果有错误，返回错误
						}
					}

					// 准备下载选项
					var options []download.Option
					// 如果用户指定了"days"选项，则添加到下载选项中
					if days := c.Int("days"); days > 0 {
						options = append(options, download.WithDays(days))
					}

					// 如果用户同时指定了"start"和"end"选项，则添加时间间隔到下载选项中
					start := c.Timestamp("start")
					end := c.Timestamp("end")
					if start != nil && end != nil && !start.IsZero() && !end.IsZero() {
						options = append(options, download.WithInterval(*start, *end))
					} else if start != nil || end != nil {
						// 如果只指定了"start"或"end"中的一个，则记录错误并退出
						log.Fatal("START and END must be informed together")
					}

					// 使用指定的选项执行下载
					return download.NewDownloader(exc).Download(c.Context, c.String("pair"),
						c.String("timeframe"), c.String("output"), options...)
				},
			},
		},
	}

	// 运行应用程序，处理命令行输入
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err) // 如果运行应用程序时发生错误，记录错误并退出
	}
}
