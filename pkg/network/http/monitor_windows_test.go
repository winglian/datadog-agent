// +build windows,npm

package http

import (
	"fmt"
	"math/rand"
	nethttp "net/http"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/network/config"
	"github.com/DataDog/datadog-agent/pkg/network/http/testutil"
	"github.com/stretchr/testify/require"
)

func TestHTTPMonitorIntegration(t *testing.T) {
	targetAddr := "localhost:8080"
	serverAddr := "localhost:8080"
	testHTTPMonitor(t, targetAddr, serverAddr, 1)
}

/*
func TestHTTPMonitorIntegrationWithNAT(t *testing.T) {
	// SetupDNAT sets up a NAT translation from 2.2.2.2 to 1.1.1.1
	netlink.SetupDNAT(t)
	defer netlink.TeardownDNAT(t)

	targetAddr := "2.2.2.2:8080"
	serverAddr := "1.1.1.1:8080"
	testHTTPMonitor(t, targetAddr, serverAddr, 10)
}
*/

func testHTTPMonitor(t *testing.T, targetAddr, serverAddr string, numReqs int) {
	srvDoneFn := testutil.HTTPServer(t, serverAddr, false)
	defer srvDoneFn()

	monitor, err := NewMonitor(config.New())
	require.NoError(t, err)

	monitor.Start()
	defer func() {
		err = monitor.Stop()
		require.NoError(t, err)
	}()
		
	// Perform a number of random requests
	requestFn := requestGenerator(t, targetAddr)
	var requests []*nethttp.Request
	for i := 0; i < numReqs; i++ {
		requests = append(requests, requestFn())
	}

	// Ensure all captured transactions get sent to user-space
	time.Sleep(10 * time.Millisecond)
	monitor.di.flushPendingTransactions()
	time.Sleep(5 * time.Millisecond)

	stats := monitor.GetHTTPStats()

	// Assert all requests made were correctly captured by the monitor
	for _, req := range requests {
		includesRequest(t, stats, req)
	}
}

func requestGenerator(t *testing.T, targetAddr string) func() *nethttp.Request {
	var (
		methods     = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		statusCodes = []int{200, 300, 400, 500}
		random      = rand.New(rand.NewSource(time.Now().Unix()))
		idx         = 0
		client      = new(nethttp.Client)
	)

	return func() *nethttp.Request {
		idx++
		method := methods[random.Intn(len(methods))]
		status := statusCodes[random.Intn(len(statusCodes))]
		url := fmt.Sprintf("http://%s/%d/request-%d", targetAddr, status, idx)
		req, err := nethttp.NewRequest(method, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		return req
	}
}

func includesRequest(t *testing.T, allStats map[Key]RequestStats, req *nethttp.Request) {
	expectedStatus := testutil.StatusFromPath(req.URL.Path)
	i := expectedStatus/100 - 1
	for key, stats := range allStats {
		t.Errorf(
		"Found transaction:\n path=%s method=%s status[%d].Count=%d. Expected:\n path=%s method=%s status=%d",
		key.Path,
		key.Method,
		expectedStatus,
		stats[i].Count,
		req.URL.Path,
		req.Method,
		expectedStatus,
		)
	}
}
