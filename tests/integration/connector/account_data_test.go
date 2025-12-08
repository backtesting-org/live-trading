package connector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Account Data Tests", func() {
	var runner *TestRunner

	BeforeEach(func() {
		var err error
		runner, err = NewTestRunner(testConnectorName, getConnectorConfig(testConnectorName))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	Context("GetAccountBalance", func() {
		It("should fetch account balance", func() {
			conn := runner.GetConnector()

			balance, err := conn.GetAccountBalance()
			AssertNoError(err, "GetAccountBalance should succeed")
			Expect(balance).ToNot(BeNil())
			Expect(balance.Currency).ToNot(BeEmpty())

			LogSuccess("Account Balance:")
			LogInfo("Currency: %s", balance.Currency)
			LogInfo("Total: %s", balance.TotalBalance.String())
			LogInfo("Available: %s", balance.AvailableBalance.String())
		})
	})

	Context("GetPositions", func() {
		It("should fetch positions", func() {
			conn := runner.GetConnector()

			positions, err := conn.GetPositions()
			AssertNoError(err, "GetPositions should succeed")
			Expect(positions).ToNot(BeNil())

			LogSuccess("Positions: %d open", len(positions))
			for i, pos := range positions {
				LogInfo("[%d] %s: Size=%s, Entry=%s", i+1,
					pos.Symbol.Symbol(), pos.Size.String(), pos.EntryPrice.String())
			}
		})
	})

	Context("GetOpenOrders", func() {
		It("should fetch open orders", func() {
			conn := runner.GetConnector()

			orders, err := conn.GetOpenOrders()
			AssertNoError(err, "GetOpenOrders should succeed")

			LogSuccess("Open orders: %d", len(orders))
		})
	})

	Context("GetTradingHistory", func() {
		It("should fetch trading history", func() {
			conn := runner.GetConnector()
			perpSymbol := conn.GetPerpSymbol(CreateAsset(testSymbol))

			history, err := conn.GetTradingHistory(perpSymbol, 10)
			AssertNoError(err, "GetTradingHistory should succeed")
			Expect(history).ToNot(BeNil())

			LogSuccess("Trading history: %d trades", len(history))
		})
	})
})
