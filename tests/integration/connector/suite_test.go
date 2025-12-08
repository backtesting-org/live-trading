package connector_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConnectorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connector Integration Suite")
}

var _ = BeforeSuite(func() {
	GinkgoWriter.Println("========================================")
	GinkgoWriter.Println("Connector Integration Test Suite")
	GinkgoWriter.Println("========================================")
})

var _ = AfterSuite(func() {
	GinkgoWriter.Println("========================================")
	GinkgoWriter.Println("Test Suite Complete")
	GinkgoWriter.Println("========================================")
})
