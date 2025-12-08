package connector_test

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Trading Operations Tests", Label("trading"), func() {
	var runner *TestRunner

	BeforeEach(func() {
		if !enableTradingTests {
			Skip("Trading tests disabled (set enableTradingTests = true to run)")
		}

		var err error
		runner, err = NewTestRunner(testConnectorName, getConnectorConfig(testConnectorName))
		Expect(err).ToNot(HaveOccurred())

		conn := runner.GetConnector()
		if !conn.SupportsTradingOperations() {
			Skip("Connector does not support trading operations")
		}
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	Context("PlaceLimitOrder", func() {
		It("should place and cancel a limit order", func() {
			conn := runner.GetConnector()
			perpSymbol := conn.GetPerpSymbol(CreateAsset(testSymbol))

			// Get current price
			priceResp, err := conn.FetchPrice(perpSymbol)
			AssertNoError(err, "FetchPrice should succeed")

			// Place order far from market to avoid fill
			limitPrice := priceResp.Price.Mul(numerical.NewFromFloat(0.5))
			qty := numerical.NewFromFloat(0.001)

			order, err := conn.PlaceLimitOrder(perpSymbol, connector.OrderSideBuy, qty, limitPrice)
			AssertNoError(err, "PlaceLimitOrder should succeed")
			Expect(order).ToNot(BeNil())
			Expect(order.OrderID).ToNot(BeEmpty())

			LogSuccess("Limit order placed: ID=%s", order.OrderID)

			// Cancel the order
			cancelResp, err := conn.CancelOrder(perpSymbol, order.OrderID)
			AssertNoError(err, "CancelOrder should succeed")
			Expect(cancelResp).ToNot(BeNil())

			LogSuccess("Order cancelled successfully")
		})
	})

	Context("GetOrderStatus", func() {
		It("should check order status", func() {
			conn := runner.GetConnector()
			perpSymbol := conn.GetPerpSymbol(CreateAsset(testSymbol))

			// Get current price and place order
			priceResp, err := conn.FetchPrice(perpSymbol)
			AssertNoError(err, "FetchPrice should succeed")

			limitPrice := priceResp.Price.Mul(numerical.NewFromFloat(0.5))
			qty := numerical.NewFromFloat(0.001)

			order, err := conn.PlaceLimitOrder(perpSymbol, connector.OrderSideBuy, qty, limitPrice)
			AssertNoError(err, "PlaceLimitOrder should succeed")

			// Check status
			status, err := conn.GetOrderStatus(order.OrderID)
			AssertNoError(err, "GetOrderStatus should succeed")
			Expect(status).ToNot(BeNil())
			Expect(status.ID).To(Equal(order.OrderID))

			LogSuccess("Order status: %s", status.Status)

			// Cleanup
			_, _ = conn.CancelOrder(perpSymbol, order.OrderID)
		})
	})
})
