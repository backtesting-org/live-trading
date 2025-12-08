package connector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocket Lifecycle Tests", func() {
	var runner *TestRunner

	BeforeEach(func() {
		var err error
		runner, err = NewTestRunner(testConnectorName, getConnectorConfig(testConnectorName))
		Expect(err).ToNot(HaveOccurred())

		if !runner.HasWebSocketSupport() {
			Skip("Connector does not support WebSocket")
		}
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	Context("StartWebSocket", func() {
		It("should establish connection", func() {
			wsConn := runner.GetWebSocketConnector()

			err := wsConn.StartWebSocket()
			AssertNoError(err, "StartWebSocket should succeed")

			Eventually(wsConn.IsWebSocketConnected, "10s", "500ms").
				Should(BeTrue(), "WebSocket should connect")

			LogSuccess("WebSocket connected")
		})
	})

	Context("StopWebSocket", func() {
		BeforeEach(func() {
			wsConn := runner.GetWebSocketConnector()
			err := wsConn.StartWebSocket()
			Expect(err).ToNot(HaveOccurred())
			Eventually(wsConn.IsWebSocketConnected, "10s").Should(BeTrue())
		})

		It("should disconnect cleanly", func() {
			wsConn := runner.GetWebSocketConnector()

			err := wsConn.StopWebSocket()
			AssertNoError(err, "StopWebSocket should succeed")

			Eventually(wsConn.IsWebSocketConnected, "5s", "500ms").
				Should(BeFalse(), "WebSocket should disconnect")

			LogSuccess("WebSocket disconnected")
		})
	})
})
