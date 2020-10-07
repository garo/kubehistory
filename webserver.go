package main

import (
	"net/http"
	"html/template"
	"github.com/gorilla/mux"
	"encoding/json"
	"fmt"
	"os"
	"github.com/golang/glog"
	"github.com/lib/pq"
	"time"
	"strings"
	"strconv"
	"net/url"
	"sort"

)

type Webserver struct {
	development bool
	Token string
	persistentStorage PersistentStorage

}

func NewWebserver(persistentStorage PersistentStorage) (*Webserver) {
	return &Webserver{
		persistentStorage: persistentStorage,
	}
}

type Page struct {
	Title string
}

type PodSnapshot struct {
	Id string `json:"id"`
	Ts string `json:"ts"`
	Reason string `json:"reason"`
	Name string `json:"name"`
	Namespace string `json:"namespace"`
	SelfLink string `json:"string"`
	Uid string `json:"uid"`
	ResourceVersion string `json:"resourceVersion"`
	CreationTimestamp string `json:"creationTimestamp"`
	DeletionTimestamp string `json:"deletionTimestamp"`
	NodeName string `json:"nodeName"`
	HostIP string `json:"hostIP"`
	PodIP string `json:"podIP"`
	Data string `json:"data"`
}

type NodeSnapshot struct {
	Id string `json:"id"`
	Ts string `json:"ts"`
	Reason string `json:"reason"`
	Name string `json:"name"`
	Namespace string `json:"namespace"`
	SelfLink string `json:"selfLink"`
	Uid string `json:"uid"`
	ResourceVersion string `json:"resourceVersion"`
	CreationTimestamp string `json:"creationTimestamp"`
	DeletionTimestamp string `json:"deletionTimestamp"`
	HostIP string `json:"hostIP"`
	Data string `json:"data"`
}

func (ws *Webserver) handleIndexPods(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	t, err := template.ParseFiles("pages/index.html")
	if err != nil {
		panic(err)
	}

	p := &Page{
		Title: "moi",
	}

	t.Execute(w, p)
}

func (ws *Webserver) handleIndexNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	t, err := template.ParseFiles("pages/nodes.html")
	if err != nil {
		panic(err)
	}

	p := &Page{
		Title: "moi",
	}

	t.Execute(w, p)
}

func (ws *Webserver) queryPods(w http.ResponseWriter, r *http.Request) {

	query := `
	SELECT id, ts, reason, name, namespace, selfLink, uid, resourceVersion,
					creationTimestamp, deletionTimestamp, nodeName, hostIP, podIP, data
	FROM pods WHERE 1=1
	`

	var args []interface{}

	if name, ok := r.URL.Query()["name"]; ok && name[0] != "" {

		if strings.Contains(name[0], "*") {
			query += fmt.Sprintf(" AND name LIKE $%d", len(args) + 1)
			n := strings.TrimSpace(name[0])
			n = strings.Replace(n, "*", "%", -1)
			args = append(args, n)
		} else {
			query += fmt.Sprintf(" AND name=$%d", len(args) + 1)
			args = append(args, strings.TrimSpace(name[0]))
		}
	}

	if podIP, ok := r.URL.Query()["podIP"]; ok && podIP[0] != "" {
		query += fmt.Sprintf(" AND podIP=$%d", len(args) + 1)
		args = append(args, strings.TrimSpace(podIP[0]))
	}

	if hostIP, ok := r.URL.Query()["hostIP"]; ok && hostIP[0] != "" {
		query += fmt.Sprintf(" AND hostIP=$%d", len(args) + 1)
		args = append(args, strings.TrimSpace(hostIP[0]))
	}

	if nodeName, ok := r.URL.Query()["hostName"]; ok && nodeName[0] != "" {
		query += fmt.Sprintf(" AND nodeName=$%d", len(args) + 1)
		args = append(args, strings.TrimSpace(nodeName[0]))
	}

	if namespace, ok := r.URL.Query()["namespace"]; ok && namespace[0] != "" {
		query += fmt.Sprintf(" AND namespace=$%d", len(args) + 1)
		args = append(args, strings.TrimSpace(namespace[0]))
	}

	query += " ORDER BY id DESC limit 500"

	glog.V(1).Infof("query: %s and arguments: %+v", query, args)
	rows, err := ws.persistentStorage.GetDB().Query(query, args[:]...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var pods []PodSnapshot

	for rows.Next() {
		var pod PodSnapshot

		var deletionTimestamp pq.NullTime

		err := rows.Scan(&pod.Id, &pod.Ts, &pod.Reason, &pod.Name, &pod.Namespace, &pod.SelfLink, &pod.Uid,
			&pod.ResourceVersion, &pod.CreationTimestamp, &deletionTimestamp, &pod.NodeName, &pod.HostIP,
			&pod.PodIP, &pod.Data)

		if deletionTimestamp.Valid {
			pod.DeletionTimestamp = deletionTimestamp.Time.Format(time.RFC3339)
		}


		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pods = append(pods, pod)
	}

	w.Header().Set("Content-Type", "application/json")
	js, err := json.MarshalIndent(pods, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)

}

func (ws *Webserver) queryNodes(w http.ResponseWriter, r *http.Request) {

	query := `
	SELECT id, ts, reason, name, selfLink, uid, resourceVersion,
					creationTimestamp, deletionTimestamp, hostIP, data
	FROM nodes WHERE 1=1
	`

	var args []interface{}

	if name, ok := r.URL.Query()["name"]; ok && name[0] != "" {

		if strings.Contains(name[0], "*") {
			query += fmt.Sprintf(" AND name LIKE $%d", len(args) + 1)
			n := strings.TrimSpace(name[0])
			n = strings.Replace(n, "*", "%", -1)
			args = append(args, n)
		} else {
			query += fmt.Sprintf(" AND name=$%d", len(args) + 1)
			args = append(args, strings.TrimSpace(name[0]))
		}
	}

	if hostIP, ok := r.URL.Query()["hostIP"]; ok && hostIP[0] != "" {
		query += fmt.Sprintf(" AND hostIP=$%d", len(args) + 1)
		args = append(args, strings.TrimSpace(hostIP[0]))
	}

	query += " ORDER BY id DESC limit 500"

	glog.V(1).Infof("query: %s and arguments: %+v", query, args)
	rows, err := ws.persistentStorage.GetDB().Query(query, args[:]...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var nodes []NodeSnapshot

	for rows.Next() {
		var node NodeSnapshot

		var deletionTimestamp pq.NullTime

		err := rows.Scan(&node.Id, &node.Ts, &node.Reason, &node.Name, &node.SelfLink, &node.Uid,
			&node.ResourceVersion, &node.CreationTimestamp, &deletionTimestamp, &node.HostIP,
			&node.Data)

		if deletionTimestamp.Valid {
			node.DeletionTimestamp = deletionTimestamp.Time.Format(time.RFC3339)
		}


		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nodes = append(nodes, node)
	}

	w.Header().Set("Content-Type", "application/json")
	js, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)

}

func (ws *Webserver) handleLogs(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "html"
	}

	labels := make(map[string]string)
	labels["namespace"] = vars["namespace"]
	labels["instance"] = vars["pod_name"]


	var start time.Time
	var end time.Time
	startUnix, err := strconv.ParseInt(vars["start"], 10, 64)
	if err == nil && startUnix != 0 {
		start = time.Unix(startUnix - 3600, 0)
	} else {
		start = time.Now().AddDate(0, 0, -1)
	}
	endUnix, err := strconv.ParseInt(vars["end"], 10, 64)
	if err == nil && endUnix != 0 {
		end = time.Unix(endUnix, 0)
	} else {
		end = time.Now()
	}

	additionalLabels := r.URL.Query()
	delete(additionalLabels, "format")
	delete(additionalLabels, "start")
	delete(additionalLabels, "end")
	delete(additionalLabels, "limit")
	delete(additionalLabels, "run_id") // It's already there, dont allow re-setting it

	for k, v := range additionalLabels {
		labels[k] = v[0]
	}

	logs, err := GetLogsByRunId(labels, start, end)
	if err != nil {
		panic(err)
	}

	uri := r.URL.RequestURI()

	additionalLabels["start"] = []string{strconv.FormatInt(start.Unix(), 10)}
	additionalLabels["end"] = []string{strconv.FormatInt(end.Unix(), 10)}
	additionalLabels["limit"] = []string{"50000"}

	glog.V(2).Infof("additionalalbels: %+v", additionalLabels)

	switch format {
	case "pretty":
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Common labels: %s\n", LogBuildLabelsString(logs.CommonLabels))
		for _, entry := range logs.Entries {
			fmt.Fprintf(w, "%s\t%s\t%s", LogBuildLabelsString(entry.Labels), entry.Ts, entry.Line)
		}
	case "html":
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<pre>\n")
		fmt.Fprintf(w, "<a href=\"/\">Back to frontpage</a>. Try also the following modes: ")
		additionalLabels["format"] = []string{"pretty"}
		fmt.Fprintf(w, "<a href=\"%s?%s\">pretty</a> ", r.URL.Path, additionalLabels.Encode())
		additionalLabels["format"] = []string{"json"}
		fmt.Fprintf(w, "<a href=\"%s?%s\">json</a> ", r.URL.Path, additionalLabels.Encode())
		additionalLabels["format"] = []string{"plain"}
		fmt.Fprintf(w, "<a href=\"%s?%s\">plain</a> ", r.URL.Path, additionalLabels.Encode())
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "Common labels: %s\n", LogBuildLabelsString(logs.CommonLabels))
		fmt.Fprintf(w, "\n")
		for _, entry := range logs.Entries {

			str := "{"
			if len(entry.Labels) > 0 {

				var keys []string

				for k, _ := range entry.Labels {
					if k == "filename" {
						continue;
					}
					keys = append(keys, k)
				}

				sort.Strings(keys)

				for _, k := range keys {
					str += fmt.Sprintf("<a href=\"%s&%s=%s\">%s=\"%s\"</a>, ", uri, k, url.QueryEscape(entry.Labels[k]), k, entry.Labels[k])
				}

				str = str[0:len(str)-2]
			}

			str += "}"

			fmt.Fprintf(w, "%s\t%s\t%s", str, entry.Ts, entry.Line)
		}
		fmt.Fprintf(w, "</pre>\n")
	case "json":
		w.Header().Set("Content-Type", "application/json")

		js, err := json.MarshalIndent(logs, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(js)
	case "plain":
		w.Header().Set("Content-Type", "text/plain")
		for _, entry := range logs.Entries {
			fmt.Fprintf(w, entry.Line)
		}
	}
}


func (ws *Webserver) handleCheck(w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "OK\n")
}

func (ws *Webserver) Start() {
	mux := mux.NewRouter()

	mux.HandleFunc("/pods", ws.queryPods)
	mux.HandleFunc("/nodes", ws.queryNodes)
	mux.HandleFunc("/", ws.handleIndexPods)
	mux.HandleFunc("/nodes.html", ws.handleIndexNodes)
	mux.HandleFunc("/healthz", ws.handleCheck)
	mux.HandleFunc("/logs/{namespace}/{pod_name}", ws.handleLogs)
	mux.PathPrefix("/static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	if _, err := os.Stat("/var/run/secrets/kubernetes.io"); os.IsNotExist(err) {
		glog.Info("Assuming running in local development env. Listening on localhost:8080")
		http.ListenAndServe("localhost:8080", mux)
		ws.development = true
	} else {
		glog.Infof("Assuming running in kubernetes. Listening on :8080")
		http.ListenAndServe(":8080", mux)
		ws.development = false
	}

}
