package exchange

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

/*
其目的是获取并更新交易对信息，并将这些信息存储在一个JSON文件中。让我们逐步分析其逻辑：
导入了一些必要的包，包括用于处理JSON数据的"encoding/json"，以及用于与Binance交易所API交互的"go-binance/v2"包。
定义了一个名为AssetQuote的结构体，用于表示交易对的报价和基础资产。
定义了一个名为pairs的变量，使用//go:embed指令嵌入了一个名为pairs.json的文件中的数据。
初始化函数init()被用来解析pairs.json文件中的交易对数据，并将其存储在一个全局映射pairAssetQuoteMap中。
SplitAssetQuote函数接收一个交易对的名称（例如 "BTCUSDT"）作为输入，并从pairAssetQuoteMap映射中查找相应的基础资产和报价资产。
updateParisFile函数用于更新本地的pairs.json文件。它执行以下操作：
创建Binance客户端实例用于访问API。
获取现货市场和期货市场的交易对信息。
遍历现货市场和期货市场的交易对信息，并将其更新到全局映射pairAssetQuoteMap中。
将pairAssetQuoteMap转换为JSON格式，并写入到pairs.json文件中。
总的逻辑是：通过Binance API获取现货市场和期货市场的交易对信息，更新全局映射pairAssetQuoteMap，并将更新后的信息写入到pairs.json文件中。
*/
// AssetQuote 结构体表示一个交易对的报价和基础资产。
type AssetQuote struct {
	Quote string // 报价资产
	Asset string // 基础资产
}

var (
	// pairs 包含 pairs.json 文件中嵌入的数据。
	//go:embed pairs.json
	pairs []byte

	// pairAssetQuoteMap 映射用来存储交易对的资产和报价。
	pairAssetQuoteMap = make(map[string]AssetQuote)
)

// 如果解析成功，pairs.json中的每个交易对和相应的资产信息将会被添加到pairAssetQuoteMap映射中。
func init() {
	err := json.Unmarshal(pairs, &pairAssetQuoteMap)
	if err != nil {
		panic(err)
	}
}

// SplitAssetQuote 函数接收一个交易对的名称（例如 "BTCUSDT"）作为输入。
// 它从全局映射 pairAssetQuoteMap 中查找与给定交易对相对应的 AssetQuote 结构体。
func SplitAssetQuote(pair string) (asset string, quote string) {
	// 从 pairAssetQuoteMap 映射中获取交易对名称对应的 AssetQuote 数据。
	data := pairAssetQuoteMap[pair]

	// 返回该数据结构中的 Asset 和 Quote 字段。
	// Asset 是基础资产（例如 "BTC"），Quote 是报价资产（例如 "USDT"）。
	return data.Asset, data.Quote
}

// updateParisFile 更新本地的 pairs.json 文件，以包含最新的交易对数据。
func updateParisFile() error {
	// 创建一个新的Binance客户端实例，用于访问API。此处"API ", "Secret "没有API密钥和秘密提供给客户端。
	client := binance.NewClient("", "")

	// 获取现货市场的交易对信息。
	sportInfo, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		// 如果获取现货市场信息失败，返回错误。
		return fmt.Errorf("failed to get exchange info: %v", err)
	}

	// 创建一个新的Binance期货客户端实例，用于访问期货市场的API。
	// 一个程序接口（API）客户端的实例，用于与Binance期货市场的API进行交互。这个客户端实例允许你的代码执行各种操作，如查询市场数据、下单、取消订单等，基于Binance期货市场的API。
	futureClient := futures.NewClient("", "")

	// 获取期货市场的交易对信息。交易对信息：每个交易对的具体信息，比如交易对标识符、允许的最小交易量、价格精度等。费率：交易和提现时可能会涉及的费用。限制：可能包括订单大小的限制、订单速率的限制等。
	futureInfo, err := futureClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		// 如果获取期货市场信息失败，返回错误。
		return fmt.Errorf("failed to get exchange info: %v", err)
	}

	// 遍历现货市场的交易对信息，并更新全局映射 pairAssetQuoteMap。
	for _, info := range sportInfo.Symbols {
		pairAssetQuoteMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset, // 报价资产
			Asset: info.BaseAsset,  // 基础资产
		}
	}

	// 遍历期货市场的交易对信息，并更新全局映射 pairAssetQuoteMap。
	for _, info := range futureInfo.Symbols {
		pairAssetQuoteMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset, // 报价资产
			Asset: info.BaseAsset,  // 基础资产
		}
	}

	// 打印出更新后映射中的交易对总数。
	fmt.Printf("Total pairs: %d\n", len(pairAssetQuoteMap))

	// 将全局映射 pairAssetQuoteMap 转换为JSON格式。
	content, err := json.Marshal(pairAssetQuoteMap)
	if err != nil {
		// 如果转换失败，返回错误。
		return fmt.Errorf("failed to marshal pairs: %v", err)
	}

	// 将JSON数据写入到 pairs.json 文件。这行代码的目的是将交易所的期货市场和现货市场的最新交易对信息
	err = os.WriteFile("pairs.json", content, 0644)
	if err != nil {
		// 如果写入文件失败，返回错误。
		return fmt.Errorf("failed to write to file: %v", err)
	}

	// 如果所有操作都成功，函数返回 nil 表示没有错误发生。
	return nil
}
