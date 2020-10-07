package main

import (
	"time"
	"encoding/json"
	//"fmt"
	"os"
	"regexp"
	"github.com/golang/glog"
	"net/url"
	"sort"
	"net/http"
	"io/ioutil"
	"strconv"
)

type LabelMap map[string]string

type LogEntry struct {
	Ts time.Time `json:"ts"`
  Line string `json:"line"`
}

type LogStream struct {
	Labels string `json:"labels"`
	Entries []LogEntry `json:"entries"`
}

type LogQueryResults struct {
	Streams []LogStream `json:"streams"`
}

type MergedLogEntry struct {
	Ts time.Time `json:"ts"`
	Line string `json:"line"`
	Labels LabelMap `"json:"labels"`
}

type MergedLogStream struct {
	CommonLabels LabelMap `json:"commonLabels"`
	Entries []MergedLogEntry `json:"entries"`
}

func LogUnmarshalQueryResults(data []byte) (LogQueryResults, error) {
	res := LogQueryResults{}
	err := json.Unmarshal(data, &res)
	return res, err
}

func MergeLogStreams(data LogQueryResults) (MergedLogStream) {

	merged := MergedLogStream{}

	numberOfStreams := len(data.Streams)

	streamIterators := make([]int, numberOfStreams)
	streamLenghts := make([]int, numberOfStreams)
	labelMaps := make([]LabelMap, numberOfStreams)
	totalLogLines := 0
	for i := range data.Streams {
		streamLenghts[i] = len(data.Streams[i].Entries)
		totalLogLines += streamLenghts[i]
		labelMaps[i] = LogParseLabels(data.Streams[i].Labels)
	}

	labelMaps, merged.CommonLabels = LogCollateLabels(labelMaps...)

	merged.Entries = make([]MergedLogEntry, totalLogLines)


	globalPosition := 0
	for true {
		toBePicked := -1
		toBePickedTs := time.Time{}
		for i, pos := range streamIterators {
			//fmt.Fprintf(os.Stderr, "iterator %d is at %d / %d\n", i, pos, streamIterators[i])

			if pos >= streamLenghts[i] {
				continue
			}

			if toBePicked == -1 {
				toBePicked = i
				toBePickedTs = data.Streams[i].Entries[pos].Ts
			} else if data.Streams[i].Entries[pos].Ts.Before(toBePickedTs) {
				toBePicked = i
				toBePickedTs = data.Streams[i].Entries[pos].Ts
			}
		}

		if toBePicked == -1 {
			//fmt.Fprintf(os.Stderr, "Line %d: Nothing to pick.\n", globalPosition)
			break
		}

		line := data.Streams[toBePicked].Entries[streamIterators[toBePicked]].Line
		ts := data.Streams[toBePicked].Entries[streamIterators[toBePicked]].Ts
		labels := labelMaps[toBePicked]

		//fmt.Fprintf(os.Stderr, "Line %d is going to be from stream %d at pos %d: \"%s\"\n", globalPosition, toBePicked, streamIterators[toBePicked], line)

		merged.Entries[globalPosition] = MergedLogEntry{
			Ts: ts,
			Line: line,
			Labels: labels,
		}

		streamIterators[toBePicked]++

		globalPosition++
	}

	return merged
}

func LogParseLabels(str string) (map[string]string) {
	labels := make(map[string]string)

	r := regexp.MustCompile("(([a-zA-Z_][a-zA-Z0-9_]*)=\"((?:[^\"\\\\]|\\\\.)*)\")")

	resp := r.FindAllStringSubmatch(str, -1)
	for _, b := range resp {
		//fmt.Fprintf(os.Stderr, "%+v is %s => %s\n", a, b[2], b[3])
		if len(b) == 4 {
			labels[b[2]] = b[3]
		} else {
			glog.Warningf("Invalid grouping in regexp for label %s. data: %+v", str, b)
		}
	}

	return labels
}


func LogCollateLabels(labelSets ...LabelMap) ([]LabelMap, LabelMap) {

	keys := make(map[string]int)
	// Find all different keys what we have
	for _, labels := range labelSets {

		// Remove some internal labels
		delete(labels, "__filename__")

		for key := range labels {
			keys[key] = keys[key] + 1
		}
	}

	//fmt.Fprintf(os.Stderr, "all keys: %+v\n", keys)

	sets := len(labelSets)

	if sets == 1 {
		return make([]LabelMap, 1), labelSets[0]
	}

	commonLabels := make(map[string]string)

	for key, count := range keys {

		// Not all labelSets share this key, so we cannot remove it in any case
		if count != sets {
			continue
		}

		v := labelSets[0][key]
		all_are_same := true
		for i := 1; i < sets; i++ {
			if v != labelSets[i][key] {
				all_are_same = false
				break
			}
		}

		if all_are_same {
			commonLabels[key] = labelSets[0][key]
			for i := range labelSets {
				delete(labelSets[i], key)
			}
		}

	}

	//fmt.Fprintf(os.Stderr, "final labelset: %+v\n", labelSets)

	return labelSets, commonLabels
}

func GetLogsByRunId(labels LabelMap, start time.Time, end time.Time) (MergedLogStream, error) {

  loki_host :=	"loki.monitoring"
	if _, err := os.Stat("/var/run/secrets/kubernetes.io"); os.IsNotExist(err) {
		loki_host = "localhost"
	}

	params := make(url.Values)

	params["start"] = []string{strconv.FormatInt(start.UnixNano(), 10)}
	params["end"] = []string{strconv.FormatInt(end.UnixNano(), 10)}
	params["direction"] = []string{"FORWARD"}

	delete(labels, "start")
	delete(labels, "end")

	promQL := LogBuildLabelsString(labels)

	url := "http://" + loki_host + ":3100/api/prom/query?" + "limit=100000&&query=" + url.QueryEscape(promQL) + "&" + params.Encode()

	glog.V(2).Infof("Getting logs as %s with url %s", promQL, url)

	resp, err := http.Get(url)

	logs := MergedLogStream{}

	if err != nil {
		return MergedLogStream{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return MergedLogStream{}, err
	}

	raw_data, err := LogUnmarshalQueryResults(body)
	if err != nil {
		return MergedLogStream{}, err
	}

	logs = MergeLogStreams(raw_data)

	return logs, nil
}

func LogBuildLabelsString(labels LabelMap) (string) {

	if len(labels) == 0 {
		return "{}"
	}
	str := "{"

	var keys []string
	//fmt.Fprintf(os.Stderr, "labels; %+v\n", labels)


	for k, _ := range labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	//fmt.Fprintf(os.Stderr, "keys after sorting: %+v\n", keys)

	for _, k := range keys {
		str += k + "=\"" + labels[k] + "\", "
	}

	str = str[0:len(str)-2]

	str += "}"

	return str
}