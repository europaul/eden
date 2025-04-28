package lim

import (
	"os"
	"testing"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/lf-edge/eve-api/go/metrics"
	"google.golang.org/protobuf/encoding/protojson"
)

type Bytes struct {
	SentByteCount int64
	RecvByteCount int64
}

var (
	fileName = "metrics_reboot3"
)

func TestSentBytes(t *testing.T) {
	data, err := os.ReadFile("testdata/" + fileName + ".json")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	metrics := metrics.ZMetricMsg{}
	err = protojson.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	urlBytes := make(map[string]Bytes)
	var bytesSentToKnownUrl int64
	for _, iface := range metrics.GetDm().Zedcloud {
		for _, urlMetic := range iface.UrlMetrics {
			bytesSentToKnownUrl += urlMetic.SentByteCount

			bytesSent := urlBytes[urlMetic.Url].SentByteCount + urlMetic.SentByteCount
			bytesRecv := urlBytes[urlMetic.Url].RecvByteCount + urlMetic.RecvByteCount
			urlBytes[urlMetic.Url] = Bytes{SentByteCount: bytesSent, RecvByteCount: bytesRecv}

		}
	}

	var totalTxBytes uint64
	for _, network := range metrics.GetDm().Network {
		totalTxBytes += network.TxBytes
	}

	// remainingBytes := totalTxBytes - uint64(bytesSentToKnownUrl)
	// bytesSentToUrl["other"] = int64(remainingBytes)

	genBarPlot(urlBytes, t)
}

func genBarPlot(urlBytes map[string]Bytes, t *testing.T) {
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title:    "Bytes Sent to URLs",
		Subtitle: "Data from metrics.json",
	}), charts.WithXAxisOpts(opts.XAxis{
		Name: "URLs",
		AxisLabel: &opts.AxisLabel{
			Rotate: 45,
		},
	}))

	urls := make([]string, 0, len(urlBytes))
	sent := make([]opts.BarData, 0, len(urlBytes))
	recv := make([]opts.BarData, 0, len(urlBytes))
	for url, bytes := range urlBytes {
		urls = append(urls, url)
		sent = append(sent, opts.BarData{Value: bytes.SentByteCount})
		recv = append(recv, opts.BarData{Value: bytes.RecvByteCount})
	}

	bar.SetXAxis(urls).
		AddSeries("Bytes Sent", sent).
		AddSeries("Bytes Recv", recv)

	f, err := os.Create(fileName + ".html")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer f.Close()

	if err := bar.Render(f); err != nil {
		t.Fatalf("failed to render chart: %v", err)
	}
}
