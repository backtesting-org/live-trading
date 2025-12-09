package connection_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConnectionManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connection Manager Suite")
}
