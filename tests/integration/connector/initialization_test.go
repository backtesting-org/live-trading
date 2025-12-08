package connector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Initialization Tests", func() {
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

	Context("when connector is registered", func() {
		It("should be initialized", func() {
			conn := runner.GetConnector()
			Expect(conn.IsInitialized()).To(BeTrue())
			LogSuccess("Connector initialized")
		})

		It("should return valid connector info", func() {
			conn := runner.GetConnector()
			info := conn.GetConnectorInfo()

			Expect(info).ToNot(BeNil())
			Expect(info.Name).To(Equal(testConnectorName))

			LogSuccess("Connector Info:")
			LogInfo("Name: %s", info.Name)
			LogInfo("Trading Enabled: %v", info.TradingEnabled)
			LogInfo("WebSocket Enabled: %v", info.WebSocketEnabled)
			LogInfo("Max Leverage: %s", info.MaxLeverage.String())
			LogInfo("Quote Currency: %s", info.QuoteCurrency)
		})

		It("should report correct capabilities", func() {
			conn := runner.GetConnector()

			_ = conn.SupportsRealTimeData()
			_ = conn.SupportsPerpetuals()
			_ = conn.SupportsSpot()
			_ = conn.SupportsFundingRates()
			_ = conn.SupportsTradingOperations()

			LogSuccess("Capabilities checked")
		})
	})
})
