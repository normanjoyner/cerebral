package influxdb

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	influxdbclient "github.com/influxdata/influxdb/client/v2"
	"github.com/influxdata/influxdb/models"

	"github.com/containership/cerebral/pkg/metrics/backends/influxdb/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
)

var (
	validURL = "http://localhost:9000"

	duration = "5m"

	goodConfiguration = map[string]string{
		"aggregation": "mean",
	}

	badAggregationConfiguration = map[string]string{
		"aggregation": "invalid-aggregation",
	}

	goodCustomQueryConfiguration = map[string]string{
		"query": "SELECT mean(\"free\") AS \"mean_free\" FROM \"telegraf\".\"rp_90d\".\"disk\" WHERE time > now() - 1m",
	}

	emptyConfiguration = map[string]string{}

	noHostnames       []string
	oneHostname       = []string{"hostname-0"}
	multipleHostnames = []string{"hostname-0", "hostname-1", "hostname-2"}
)

func TestNewClient(t *testing.T) {
	// Should never fail with any valid URL because it's only constructing an
	// http.Client under the hood
	client, err := NewClient(validURL, corelistersv1.NewNodeLister(nil))
	assert.NotNil(t, client)
	assert.NoError(t, err, "any valid URL is ok")

	_, err = NewClient("", corelistersv1.NewNodeLister(nil))
	assert.Error(t, err, "error on empty URL")

	_, err = NewClient(validURL, nil)
	assert.Error(t, err, "error on nil NodeLister")
}

func TestGetValue(t *testing.T) {
	nodeLister := buildNodeLister(nil)

	mockInfluxDB := mocks.Client{}
	// Return error
	mockInfluxDB.On("Query", mock.Anything).
		Return(nil, fmt.Errorf("some InfluxDB error")).Once()

	backend := Backend{
		influxDB:   &mockInfluxDB,
		nodeLister: nodeLister,
	}

	_, err := backend.GetValue("cpu_percent_utilization", goodConfiguration, nil)
	assert.Error(t, err, "error when InfluxDB errors")

	// Return unexpected nil
	mockInfluxDB.On("Query", mock.Anything).
		Return(nil, nil).Once()

	_, err = backend.GetValue("cpu_percent_utilization", goodConfiguration, nil)
	assert.Error(t, err, "error on nil result")

	// Return unexpected non-Vector type
	mockInfluxDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
		Return(&influxdbclient.Response{
			Err: "Influxdb error response",
		}, nil).Once()

	_, err = backend.GetValue("cpu_percent_utilization", goodConfiguration, nil)
	assert.Error(t, err, "error on query response")

	// Return single element vector as expected
	mockInfluxDB.On("Query", mock.Anything).
		Return(&influxdbclient.Response{
			Results: []influxdbclient.Result{
				{
					Series: []models.Row{
						{
							Name:    "cpu",
							Columns: []string{"time", "cpu_percent_utilization"},
							Values: [][]interface{}{
								{"2018-12-25T16:12:06.249608977Z", json.Number("36.555302259839486")},
							},
						},
					},
				},
			},
		}, nil).Once()

	_, err = backend.GetValue("cpu_percent_utilization", goodConfiguration, nil)
	assert.NoError(t, err, "single element vector is ok")
}

func TestBuildCPUQuery(t *testing.T) {
	_, err := buildCPUQuery(oneHostname, goodConfiguration)
	assert.NoError(t, err, "good configuration is ok")

	_, err = buildCPUQuery(oneHostname, emptyConfiguration)
	assert.NoError(t, err, "empty configuration is ok (defaults)")

	_, err = buildCPUQuery(oneHostname, badAggregationConfiguration)
	assert.Error(t, err, "invalid aggregation errors")
}

func TestBuildMemoryQuery(t *testing.T) {
	_, err := buildMemoryQuery(oneHostname, goodConfiguration)
	assert.NoError(t, err, "good configuration is ok")

	_, err = buildMemoryQuery(oneHostname, emptyConfiguration)
	assert.NoError(t, err, "empty configuration is ok (defaults)")

	_, err = buildMemoryQuery(oneHostname, badAggregationConfiguration)
	assert.Error(t, err, "invalid aggregation errors")
}

func TestBuildCustomQuery(t *testing.T) {
	_, err := buildCustomQuery(oneHostname, goodCustomQueryConfiguration)
	assert.NoError(t, err, "good configuration is ok")

	_, err = buildCustomQuery(oneHostname, emptyConfiguration)
	assert.Error(t, err, "empty configuration is invalid")
}

func TestBuildHostList(t *testing.T) {
	hostList := buildHostList(nil)
	assert.Equal(t, "(true)", hostList, "nil hostnames results in (true)")

	hostList = buildHostList(noHostnames)
	assert.Equal(t, "(true)", hostList, "no hostnames results in (true)")

	hostList = buildHostList(oneHostname)
	assert.Equal(t, "(\"host\"='hostname-0')", hostList, "single hostname hostList")

	hostList = buildHostList(multipleHostnames)
	assert.Equal(t, "(\"host\"='hostname-0' OR \"host\"='hostname-1' OR \"host\"='hostname-2')", hostList, "multiple hostname hostList")
}

// Get a node lister. Copies of the nodes are added to the cache; not the nodes themselves.
func buildNodeLister(nodes []corev1.Node) corelistersv1.NodeLister {
	// We don't need anything related to the client or informer; we're simply
	// using this as an easy way to build a cache
	client := &fake.Clientset{}
	kubeInformerFactory := informers.NewSharedInformerFactory(client, 30*time.Second)
	informer := kubeInformerFactory.Core().V1().Nodes()

	for _, node := range nodes {
		// TODO why is DeepCopy() required here? Without it, each Add() duplicates
		// the first member added.
		err := informer.Informer().GetStore().Add(node.DeepCopy())
		if err != nil {
			// Should be a programming error
			panic(err)
		}
	}

	return informer.Lister()
}
