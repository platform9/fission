package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wcharczuk/go-chart"
)

var (
	title        *string
	file         *string
	outputFile   *string
	outputFormat *string
)

const (
	PNG = "png"
	SVG = "svg"
)

type (
	Tags struct {
		Group  string `json:"group"`
		Iter   string `json:"iter"`
		Method string `json:"method"`
		Name   string `json:"name"`
		Proto  string `json:"proto"`
		Status string `json:"status"`
		URL    string `json:"url"`
		Vu     string `json:"vu"`
	}

	Data struct {
		Time  time.Time `json:"time"`
		Value float64   `json:"value"`
		Tags  Tags      `json:"tags"`
	}

	MetricPoint struct {
		Type   string `json:"type"`
		Data   Data   `json:"data"`
		Metric string `json:"metric"`
	}
)

func init() {
	title = flag.String("title", "Fission Benchmark", "Chart title")
	file = flag.String("file", "", "Metric json file")
	outputFile = flag.String("o", "chart.png", "Output file name")
	outputFormat = flag.String("format", PNG, "Format of output file (png or svg)")
	flag.Parse()
}

func generateContinuousSeries(file string) chart.Series {
	f, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open file metric point: %v", err)
		return nil
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	var xVals []float64
	var yVals []float64

	var initTime *time.Time

	for {
		l, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}

		point := &MetricPoint{}
		err = json.Unmarshal(l, point)
		if err != nil {
			fmt.Printf("Failed to parse metric point: %v -> %v", err, string(l))
			return nil
		}

		if point.Type != "Point" || point.Metric != "http_req_duration" {
			continue
		}

		if initTime == nil {
			initTime = &point.Data.Time
		}

		timeSinceStart := point.Data.Time.Sub(*initTime).Seconds()
		xVals = append(xVals, timeSinceStart)
		yVals = append(yVals, point.Data.Value)
	}

	return chart.ContinuousSeries{
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.GetDefaultColor(0).WithAlpha(64),
			FillColor:   chart.GetDefaultColor(0).WithAlpha(64),
		},
		XValues: xVals,
		YValues: yVals,
	}
}

func generateChart(title string, file string, format chart.RendererProvider, series []chart.Series) error {
	if series == nil {
		return errors.New("series cannot be nil")
	}

	cs := chart.ConcatSeries(series)

	graph := chart.Chart{
		Title:      title,
		TitleStyle: chart.StyleShow(),
		Background: chart.Style{
			Padding: chart.Box{
				Top:    50,
				Left:   25,
				Right:  25,
				Bottom: 10,
			},
		},
		XAxis: chart.XAxis{
			Name:      "Time (s)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			Range: &chart.ContinuousRange{
				Min: 0,
			},
			ValueFormatter: func(v interface{}) string {
				return fmt.Sprintf("%.2f s", v.(float64))
			},
		},
		YAxis: chart.YAxis{
			Name:      "Response Time (ms)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			Range: &chart.ContinuousRange{
				Min: 0,
			},
			ValueFormatter: func(v interface{}) string {
				return fmt.Sprintf("%d ms", int(v.(float64)))
			},
		},
		Series: cs,
	}

	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(format, buffer)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, buffer.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func listJsonFiles(path string) ([]string, error) {
	fi, err := os.Stat(*file)
	if err != nil {
		return nil, err
	}

	if fi.Mode().IsRegular() {
		return []string{path}, nil
	}

	var files []string

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".json") && !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func main() {

	if file == nil || len(*file) == 0 {
		fmt.Println("Please provide metric json file name")
		return
	}

	var format chart.RendererProvider

	if outputFormat == nil {
		format = chart.PNG
	} else {
		switch strings.ToLower(*outputFormat) {
		case PNG:
			format = chart.PNG
		case SVG:
			format = chart.SVG
		default:
			fmt.Println("Unknown format, use png as output format")
			*outputFormat = PNG
		}
	}

	if len(*outputFile) == 0 {
		fmt.Println("Please output chart png file name")
		return
	}

	files, err := listJsonFiles(*file)
	if err != nil {
		fmt.Printf("Failed to get file information: %v", err)
		return
	}

	series := make([]chart.Series, len(files))
	for i, f := range files {
		series[i] = generateContinuousSeries(f)
	}

	err = generateChart(*title, *outputFile, format, series)
	if err != nil {
		fmt.Printf("Failed to generate chart: %v\n", err)
	}
}
