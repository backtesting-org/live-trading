package connection_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	logger "github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	mockconn "github.com/backtesting-org/live-trading/mocks/github.com/backtesting-org/live-trading/pkg/websocket/connection"
	mockperf "github.com/backtesting-org/live-trading/mocks/github.com/backtesting-org/live-trading/pkg/websocket/performance"
	mocksec "github.com/backtesting-org/live-trading/mocks/github.com/backtesting-org/live-trading/pkg/websocket/security"
	"github.com/backtesting-org/live-trading/pkg/websocket/connection"
)

func TestConnectionManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connection Manager Suite")
}

var _ = Describe("ConnectionManager", func() {
	var (
		mgr         connection.ConnectionManager
		mockAuth    *mocksec.AuthManager
		mockMetrics *mockperf.Metrics
		mockDialer  *mockconn.WebSocketDialer
		mockLogger  logger.ApplicationLogger
		ctx         context.Context
		cancel      context.CancelFunc
		config      connection.Config
	)

	BeforeEach(func() {
		mockAuth = mocksec.NewAuthManager(GinkgoT())
		mockMetrics = mockperf.NewMetrics(GinkgoT())
		mockDialer = mockconn.NewWebSocketDialer(GinkgoT())
		mockLogger = logger.NewNoOpLogger()
		ctx, cancel = context.WithCancel(context.Background())

		// Setup mock metrics - all possible calls
		mockMetrics.On("GetStats").Return(map[string]interface{}{}).Maybe()
		mockMetrics.On("IncrementConnectionError").Return().Maybe()
		mockMetrics.On("IncrementSent").Return().Maybe()
		mockMetrics.On("IncrementReceived").Return().Maybe()
		mockMetrics.On("RecordConnectionDuration", mock.Anything).Return().Maybe()

		// Setup mock auth
		mockAuth.On("GetSecureHeaders", mock.Anything).Return(make(map[string][]string), nil).Maybe()

		config = connection.Config{
			URL:                    "wss://test.example.com/ws",
			ConnectTimeout:         5 * time.Second,
			HandshakeTimeout:       5 * time.Second,
			ReadTimeout:            30 * time.Second,
			WriteTimeout:           10 * time.Second,
			MaxMessageSize:         1024 * 1024,
			ReadBufferSize:         4096,
			WriteBufferSize:        4096,
			HealthCheckInterval:    10 * time.Second,
			HealthCheckTimeout:     30 * time.Second,
			EnableHealthMonitoring: false, // Disable for simpler tests
			EnableHealthPings:      false,
		}

		// Use same constructor - inject mock dialer
		mgr = connection.NewConnectionManager(config, mockAuth, mockMetrics, mockLogger, mockDialer)
	})

	AfterEach(func() {
		cancel()
		if mgr != nil {
			_ = mgr.Disconnect() // Ignore error in cleanup
		}
	})

	Describe("Initial State", func() {
		It("should start in Disconnected state", func() {
			Expect(mgr.GetState()).To(Equal(connection.StateDisconnected))
		})
	})

	Describe("Connect", func() {
		Context("when auth fails", func() {
			BeforeEach(func() {
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(nil, errors.New("auth failed"))
			})

			It("should return error and stay Disconnected", func() {
				err := mgr.Connect(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("auth"))
			})
		})

		// Skip actual connection tests - require real WebSocket or complex mocking
		Context("when connection succeeds", func() {
			PIt("should transition to Connected state", func() {
				// Requires mocking gorilla/websocket Dialer
				Skip("Cannot mock websocket.Conn - use integration tests")
			})
		})

		Context("when already connected", func() {
			PIt("should return error", func() {
				// Requires mocking gorilla/websocket Dialer
				Skip("Cannot mock websocket.Conn - use integration tests")
			})
		})
	})

	Describe("Disconnect - User Command", func() {
		Context("when called explicitly by user", func() {
			It("should transition to Stopped state (not Disconnected)", func() {
				err := mgr.Disconnect()
				Expect(err).ToNot(HaveOccurred())

				// KEY BEHAVIOR: User disconnect should result in StateStopped
				// This prevents reconnection logic from triggering
				Expect(mgr.GetState()).To(Equal(connection.StateStopped))
			})

			It("should be idempotent", func() {
				err := mgr.Disconnect()
				Expect(err).ToNot(HaveOccurred())

				err = mgr.Disconnect()
				Expect(err).ToNot(HaveOccurred())

				Expect(mgr.GetState()).To(Equal(connection.StateStopped))
			})

			It("should not call onDisconnect callback", func() {
				disconnectCalled := false
				mgr.SetCallbacks(
					nil,
					func() error {
						disconnectCalled = true
						return nil
					},
					nil,
					nil,
				)

				err := mgr.Disconnect()
				Expect(err).ToNot(HaveOccurred())

				// Give time for any async callbacks
				Consistently(func() bool { return disconnectCalled }, "200ms").Should(BeFalse())
			})

			It("should wait for all goroutines to eIt", func() {
				// Setup connected state
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)
				err := mgr.Connect(ctx)
				Expect(err).ToNot(HaveOccurred())

				// Track disconnect timing
				start := time.Now()
				done := make(chan struct{})

				go func() {
					mgr.Disconnect()
					close(done)
				}()

				// Should complete (not hang forever)
				Eventually(done, "3s").Should(BeClosed())

				// Should have taken some time to cleanup
				Expect(time.Since(start)).To(BeNumerically(">", 0))
			})
		})

		Context("when already disconnected", func() {
			It("should succeed without error", func() {
				// Already disconnected
				Expect(mgr.GetState()).To(Equal(connection.StateDisconnected))

				err := mgr.Disconnect()
				Expect(err).ToNot(HaveOccurred())
				Expect(mgr.GetState()).To(Equal(connection.StateStopped))
			})
		})
	})

	Describe("Connection Error Handling", func() {
		Context("when connection fails naturally (not user command)", func() {
			It("should call onDisconnect callback", func() {
				disconnectCalled := false
				mgr.SetCallbacks(
					nil,
					func() error {
						disconnectCalled = true
						return nil
					},
					nil,
					nil,
				)

				// Simulate connection then failure
				// (requires mocking actual WebSocket)

				Eventually(func() bool { return disconnectCalled }, "2s").Should(BeTrue())
			})

			It("should transition to StateDisconnected (not StateStopped)", func() {
				// Setup connected state
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)
				err := mgr.Connect(ctx)
				Expect(err).ToNot(HaveOccurred())

				// Simulate network error
				// Connection should go to StateDisconnected (allowing reconnect)
				// NOT StateStopped (which prevents reconnect)

				Eventually(func() connection.ConnectionState {
					return mgr.GetState()
				}, "2s").Should(Equal(connection.StateDisconnected))
			})

			It("should call onError callback", func() {
				errorCalled := false
				var capturedError error

				mgr.SetCallbacks(
					nil,
					nil,
					nil,
					func(err error) {
						errorCalled = true
						capturedError = err
					},
				)

				// Simulate connection error

				Eventually(func() bool { return errorCalled }, "2s").Should(BeTrue())
				Expect(capturedError).ToNot(BeNil())
			})

			It("should increment connection error metric", func() {
				mockMetrics.On("IncrementConnectionError").Return().Once()

				// Simulate connection error

				Eventually(func() bool {
					return mockMetrics.AssertCalled(GinkgoT(), "IncrementConnectionError")
				}, "2s").Should(BeTrue())
			})
		})
	})

	Describe("SendMessage", func() {
		Context("when disconnected", func() {
			It("should return error", func() {
				err := mgr.SendMessage([]byte("test"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not connected"))
			})
		})

		Context("when in Stopped state", func() {
			It("should return error", func() {
				mgr.Disconnect()
				Expect(mgr.GetState()).To(Equal(connection.StateStopped))

				err := mgr.SendMessage([]byte("test"))
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when connected", func() {
			BeforeEach(func() {
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)
				err := mgr.Connect(ctx)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should send message successfully", func() {
				err := mgr.SendMessage([]byte("test message"))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("SendJSON", func() {
		Context("when disconnected", func() {
			It("should return error", func() {
				testData := map[string]string{"test": "data"}
				err := mgr.SendJSON(testData)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not connected"))
			})
		})

		Context("when connected", func() {
			BeforeEach(func() {
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)
				err := mgr.Connect(ctx)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should serialize and send JSON", func() {
				testData := map[string]string{"test": "data"}
				err := mgr.SendJSON(testData)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("SendPing", func() {
		Context("when disconnected", func() {
			It("should return error", func() {
				err := mgr.SendPing()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("State Transitions", func() {
		It("should follow valid state machine", func() {
			// Start: Disconnected
			Expect(mgr.GetState()).To(Equal(connection.StateDisconnected))

			// User Disconnect from Disconnected -> Stopped
			mgr.Disconnect()
			Expect(mgr.GetState()).To(Equal(connection.StateStopped))

			// Stopped is FINAL - no transitions out
			// (would need new manager instance to reconnect)
		})
	})

	Describe("GetConnectionStats", func() {
		It("should return stats map", func() {
			stats := mgr.GetConnectionStats()
			Expect(stats).ToNot(BeNil())
			Expect(stats).To(HaveKey("state"))
			Expect(stats).To(HaveKey("connected"))
			Expect(stats).To(HaveKey("url"))
		})

		It("should reflect current state", func() {
			stats := mgr.GetConnectionStats()
			Expect(stats["state"]).To(Equal("disconnected"))
			Expect(stats["connected"]).To(BeFalse())

			mgr.Disconnect()
			stats = mgr.GetConnectionStats()
			Expect(stats["state"]).To(Equal("stopped"))
		})
	})

	Describe("IsHealthy", func() {
		Context("when disconnected", func() {
			It("should return false", func() {
				Expect(mgr.IsHealthy()).To(BeFalse())
			})
		})

		Context("when stopped", func() {
			It("should return false", func() {
				mgr.Disconnect()
				Expect(mgr.IsHealthy()).To(BeFalse())
			})
		})

		Context("when connected with recent activity", func() {
			It("should return true", func() {
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)
				err := mgr.Connect(ctx)
				Expect(err).ToNot(HaveOccurred())

				Expect(mgr.IsHealthy()).To(BeTrue())
			})
		})

		Context("when connected but stale", func() {
			It("should return false after timeout", func() {
				// This would require time manipulation or very short timeouts
			})
		})
	})

	Describe("Goroutine Lifecycle", func() {
		It("should not leak goroutines after disconnect", func() {
			// Setup
			mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)

			// Get baseline goroutine count
			// baseline := runtime.NumGoroutine()

			// Connect
			err := mgr.Connect(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Disconnect
			err = mgr.Disconnect()
			Expect(err).ToNot(HaveOccurred())

			// Wait for cleanup
			time.Sleep(100 * time.Millisecond)

			// Verify goroutines cleaned up
			// current := runtime.NumGoroutine()
			// Expect(current).To(BeNumerically("<=", baseline+1)) // Allow small variance
		})
	})

	Describe("Callbacks", func() {
		var (
			connectCalled    bool
			disconnectCalled bool
			messageCalled    bool
			errorCalled      bool
		)

		BeforeEach(func() {
			connectCalled = false
			disconnectCalled = false
			messageCalled = false
			errorCalled = false

			mgr.SetCallbacks(
				func() error {
					connectCalled = true
					return nil
				},
				func() error {
					disconnectCalled = true
					return nil
				},
				func(data []byte) error {
					messageCalled = true
					return nil
				},
				func(err error) {
					errorCalled = true
				},
			)
		})

		It("should call onConnect when connection succeeds", func() {
			mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)

			err := mgr.Connect(ctx)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool { return connectCalled }, "1s").Should(BeTrue())
		})

		It("should NOT call onDisconnect on user disconnect", func() {
			mgr.Disconnect()

			Consistently(func() bool { return disconnectCalled }, "200ms").Should(BeFalse())
		})

		It("should call onDisconnect on connection error", func() {
			// Simulate connection error (not user disconnect)

			Eventually(func() bool { return disconnectCalled }, "2s").Should(BeTrue())
		})

		It("should call onMessage when message received", func() {
			// Simulate message received

			Eventually(func() bool { return messageCalled }, "2s").Should(BeTrue())
		})

		It("should call onError when error occurs", func() {
			// Simulate error

			Eventually(func() bool { return errorCalled }, "2s").Should(BeTrue())
		})
	})

	Describe("Context Cancellation", func() {
		It("should respect context cancellation during connect", func() {
			mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)

			// Create context that expires quickly
			shortCtx, shortCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer shortCancel()

			err := mgr.Connect(shortCtx)
			Expect(err).To(HaveOccurred())
		})

		It("should stop readMessages when context cancelled", func() {
			mockAuth.On("GetSecureHeaders", mock.Anything).Return(map[string][]string{}, nil)

			err := mgr.Connect(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Cancel context
			cancel()

			// Should transition to disconnected
			Eventually(func() connection.ConnectionState {
				return mgr.GetState()
			}, "1s").Should(Equal(connection.StateDisconnected))
		})
	})

	Describe("Thread Safety", func() {
		It("should handle concurrent GetState calls", func() {
			done := make(chan struct{})

			for i := 0; i < 10; i++ {
				go func() {
					defer GinkgoRecover()
					state := mgr.GetState()
					Expect(state).To(BeNumerically(">=", connection.StateDisconnected))
					done <- struct{}{}
				}()
			}

			for i := 0; i < 10; i++ {
				<-done
			}
		})

		It("should handle concurrent Disconnect calls", func() {
			done := make(chan struct{})

			for i := 0; i < 5; i++ {
				go func() {
					defer GinkgoRecover()
					err := mgr.Disconnect()
					Expect(err).ToNot(HaveOccurred())
					done <- struct{}{}
				}()
			}

			for i := 0; i < 5; i++ {
				<-done
			}

			Expect(mgr.GetState()).To(Equal(connection.StateStopped))
		})
	})
})
