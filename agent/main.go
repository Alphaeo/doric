package main

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── Types ─────────────────────────────────────────────────────────────────

type DeployRequest struct {
	ProjectID string            `json:"project_id"`
	RepoURL   string            `json:"repo_url"`
	Branch    string            `json:"branch"`
	Domain    string            `json:"domain"`
	EnvVars   map[string]string `json:"env_vars"`
}

type LogLine struct {
	Log     string `json:"log"`
	Done    bool   `json:"done"`
	Success bool   `json:"success,omitempty"`
	AppURL  string `json:"app_url,omitempty"`
}

type ControlRequest struct {
	ContainerName string `json:"container_name"`
	Action        string `json:"action"` // start | stop | restart | remove
}

// RunRequest — lance une image Docker existante (pas de build)
type RunRequest struct {
	ContainerName string            `json:"container_name"`
	Image         string            `json:"image"`
	EnvVars       map[string]string `json:"env_vars"`
	Ports         map[string]string `json:"ports"`  // hostPort → containerPort
	Network       string            `json:"network"`
	Volumes       map[string]string `json:"volumes"` // hostPath → containerPath
}

type ContainerInfo struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
	ID     string `json:"id"`
}

// ── Helpers ───────────────────────────────────────────────────────────────

// sender permet de streamer des lignes de log JSON vers le client HTTP.
type sender struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSender(w http.ResponseWriter) (*sender, bool) {
	f, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	return &sender{w: w, flusher: f}, true
}

func (s *sender) log(msg string) {
	line, _ := json.Marshal(LogLine{Log: msg})
	fmt.Fprintf(s.w, "%s\n", line)
	s.flusher.Flush()
}

func (s *sender) done(success bool, appURL string) {
	line, _ := json.Marshal(LogLine{Done: true, Success: success, AppURL: appURL})
	fmt.Fprintf(s.w, "%s\n", line)
	s.flusher.Flush()
}

// streamCmd exécute une commande et envoie chaque ligne de sortie au client.
func streamCmd(ctx context.Context, s *sender, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = cmd.Stdout // merge stderr in stdout pipe

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		s.log(scanner.Text())
	}

	return cmd.Wait()
}

// ── Handlers ──────────────────────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.ProjectID == "" || req.RepoURL == "" {
		http.Error(w, "project_id and repo_url are required", http.StatusBadRequest)
		return
	}

	s, ok := newSender(w)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	branch := req.Branch
	if branch == "" {
		branch = "main"
	}
	domain := req.Domain
	if domain == "" {
		domain = req.ProjectID + ".localhost"
	}
	workDir := filepath.Join("/tmp/doric", req.ProjectID)
	imageName := "doric-" + req.ProjectID
	containerName := "doric-app-" + req.ProjectID

	// ── 1. Git clone ──────────────────────────────────────────────────────
	s.log(fmt.Sprintf("[git] Cloning %s @ %s ...", req.RepoURL, branch))
	os.RemoveAll(workDir)

	if err := streamCmd(ctx, s, "git", "clone", "--depth=1", "--branch", branch, req.RepoURL, workDir); err != nil {
		s.log("[git] Error: " + err.Error())
		s.done(false, "")
		return
	}
	s.log("[git] Clone OK")

	// ── 2. Docker build ───────────────────────────────────────────────────
	s.log(fmt.Sprintf("[docker] Building image %s ...", imageName))
	if err := streamCmd(ctx, s, "docker", "build", "--no-cache", "-t", imageName, workDir); err != nil {
		s.log("[docker] Build error: " + err.Error())
		s.done(false, "")
		return
	}
	s.log("[docker] Build OK")

	// ── 3. Arrêt du conteneur précédent ──────────────────────────────────
	exec.Command("docker", "stop", containerName).Run()
	exec.Command("docker", "rm", containerName).Run()

	// ── 4. Docker run avec labels Traefik ────────────────────────────────
	s.log(fmt.Sprintf("[docker] Starting container %s ...", containerName))

	args := []string{
		"run", "-d",
		"--name", containerName,
		"--restart", "unless-stopped",
		"--network", "doric",
		"--label", "traefik.enable=true",
		"--label", fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s`)", req.ProjectID, domain),
		"--label", fmt.Sprintf("traefik.http.routers.%s.entrypoints=web,websecure", req.ProjectID),
		"--label", fmt.Sprintf("traefik.http.routers.%s.tls.certresolver=letsencrypt", req.ProjectID),
		"--label", fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=8080", req.ProjectID),
	}

	for k, v := range req.EnvVars {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, imageName)

	if err := streamCmd(ctx, s, "docker", args...); err != nil {
		s.log("[docker] Run error: " + err.Error())
		s.done(false, "")
		return
	}

	appURL := "https://" + domain
	s.log("[doric] Deployment complete!")
	s.log("[doric] App live at " + appURL)
	s.done(true, appURL)
}

func handleControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var cmd *exec.Cmd
	switch req.Action {
	case "start":
		cmd = exec.Command("docker", "start", req.ContainerName)
	case "stop":
		cmd = exec.Command("docker", "stop", req.ContainerName)
	case "restart":
		cmd = exec.Command("docker", "restart", req.ContainerName)
	case "remove":
		exec.Command("docker", "stop", req.ContainerName).Run()
		cmd = exec.Command("docker", "rm", req.ContainerName)
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	out, err := cmd.CombinedOutput()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": strings.TrimSpace(string(out)) + ": " + err.Error(),
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": strings.TrimSpace(string(out)),
	})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	containerName := r.URL.Query().Get("container")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "100"
	}
	follow := r.URL.Query().Get("follow") == "true"

	if containerName == "" {
		http.Error(w, "container query param required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	args := []string{"logs", "--tail", tail}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, containerName)

	cmd := exec.CommandContext(r.Context(), "docker", args...)
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Run()
}

// ── MinIO S3 helper (Signature V4) ───────────────────────────────────────

func minioEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func minioEndpoint() string {
	ep := minioEnv("MINIO_ENDPOINT", "localhost:9000")
	if strings.HasPrefix(ep, "http") {
		return ep
	}
	return "http://" + ep
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sha256hex(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func minioRequest(method, path, query string, body io.Reader) (*http.Response, error) {
	accessKey := minioEnv("MINIO_ROOT_USER", "doric")
	secretKey := minioEnv("MINIO_ROOT_PASSWORD", "doricseed")
	region    := "us-east-1"

	t       := time.Now().UTC()
	dateISO  := t.Format("20060102T150405Z")
	dateShort:= t.Format("20060102")

	payload  := sha256hex("")
	host     := strings.TrimPrefix(strings.TrimPrefix(minioEndpoint(), "https://"), "http://")

	canonHeaders := "host:" + host + "\nx-amz-content-sha256:" + payload + "\nx-amz-date:" + dateISO + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonReq := strings.Join([]string{method, path, query, canonHeaders, signedHeaders, payload}, "\n")
	credScope := strings.Join([]string{dateShort, region, "s3", "aws4_request"}, "/")
	strToSign := "AWS4-HMAC-SHA256\n" + dateISO + "\n" + credScope + "\n" + sha256hex(canonReq)

	sigKey := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256([]byte("AWS4"+secretKey), dateShort), region), "s3"), "aws4_request")
	sig    := hex.EncodeToString(hmacSHA256(sigKey, strToSign))

	authHdr := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s,SignedHeaders=%s,Signature=%s",
		accessKey, credScope, signedHeaders, sig)

	url := minioEndpoint() + path
	if query != "" {
		url += "?" + query
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Amz-Date", dateISO)
	req.Header.Set("X-Amz-Content-Sha256", payload)
	req.Header.Set("Authorization", authHdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	return http.DefaultClient.Do(req)
}

// ── /stats — métriques containers docker ─────────────────────────────────

type ContainerStat struct {
	Name     string `json:"name"`
	CPU      string `json:"cpu"`
	MemUsage string `json:"mem_usage"`
	MemPerc  string `json:"mem_perc"`
	NetIO    string `json:"net_io"`
	PIDs     string `json:"pids"`
	Status   string `json:"status"`
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	filter := r.URL.Query().Get("filter")

	out, err := exec.Command("docker", "stats", "--no-stream", "--format",
		`{"name":"{{.Name}}","cpu":"{{.CPUPerc}}","mem_usage":"{{.MemUsage}}","mem_perc":"{{.MemPerc}}","net_io":"{{.NetIO}}","pids":"{{.PIDs}}"}`).Output()
	if err != nil {
		json.NewEncoder(w).Encode([]ContainerStat{})
		return
	}

	var stats []ContainerStat
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var s ContainerStat
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue
		}
		if filter != "" && !strings.Contains(s.Name, filter) {
			continue
		}
		// Only doric-* containers if no filter
		if filter == "" && !strings.HasPrefix(s.Name, "doric-") {
			continue
		}
		s.Status = "running"
		stats = append(stats, s)
	}
	if stats == nil {
		stats = []ContainerStat{}
	}
	json.NewEncoder(w).Encode(stats)
}

// ── /storage/* — MinIO S3 ─────────────────────────────────────────────────

// Réponse XML de ListAllMyBuckets
type s3BucketList struct {
	Buckets []struct {
		Name         string `xml:"Name"`
		CreationDate string `xml:"CreationDate"`
	} `xml:"Buckets>Bucket"`
}

// Réponse XML de ListObjectsV2
type s3ObjectList struct {
	Contents []struct {
		Key          string `xml:"Key"`
		Size         int64  `xml:"Size"`
		LastModified string `xml:"LastModified"`
		ETag         string `xml:"ETag"`
	} `xml:"Contents"`
	Name        string `xml:"Name"`
	IsTruncated bool   `xml:"IsTruncated"`
}

func handleStorageBuckets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List buckets
		resp, err := minioRequest("GET", "/", "", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		defer resp.Body.Close()
		var list s3BucketList
		xml.NewDecoder(resp.Body).Decode(&list)
		type bucket struct {
			Name    string `json:"name"`
			Created string `json:"created"`
		}
		buckets := []bucket{}
		for _, b := range list.Buckets {
			buckets = append(buckets, bucket{Name: b.Name, Created: b.CreationDate})
		}
		json.NewEncoder(w).Encode(buckets)

	case http.MethodPost:
		// Create bucket
		var req struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		resp, err := minioRequest("PUT", "/"+req.Name, "", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		resp.Body.Close()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.StatusCode == 200 || resp.StatusCode == 409})

	case http.MethodDelete:
		bucket := r.URL.Query().Get("bucket")
		if bucket == "" {
			http.Error(w, "bucket required", http.StatusBadRequest)
			return
		}
		resp, err := minioRequest("DELETE", "/"+bucket, "", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		resp.Body.Close()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.StatusCode == 204})
	}
}

func handleStorageObjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		http.Error(w, "bucket required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		resp, err := minioRequest("GET", "/"+bucket, "list-type=2&max-keys=1000", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		defer resp.Body.Close()
		var list s3ObjectList
		xml.NewDecoder(resp.Body).Decode(&list)
		type obj struct {
			Key     string `json:"key"`
			Size    int64  `json:"size"`
			Date    string `json:"date"`
			ETag    string `json:"etag"`
		}
		objects := []obj{}
		for _, o := range list.Contents {
			objects = append(objects, obj{Key: o.Key, Size: o.Size, Date: o.LastModified, ETag: strings.Trim(o.ETag, `"`)})
		}
		json.NewEncoder(w).Encode(objects)

	case http.MethodDelete:
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key required", http.StatusBadRequest)
			return
		}
		resp, err := minioRequest("DELETE", "/"+bucket+"/"+key, "", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		resp.Body.Close()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.StatusCode == 204})
	}
}

func handleStorageUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	bucket := r.URL.Query().Get("bucket")
	key    := r.URL.Query().Get("key")
	if bucket == "" || key == "" {
		http.Error(w, "bucket and key required", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	resp, err := minioRequest("PUT", "/"+bucket+"/"+key, "", r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	resp.Body.Close()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.StatusCode == 200,
		"url":     minioEndpoint() + "/" + bucket + "/" + key,
	})
}

// ── handleRun — lance un container depuis une image existante (BDD, services)
func handleRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ContainerName == "" || req.Image == "" {
		http.Error(w, "container_name and image are required", http.StatusBadRequest)
		return
	}

	// Arrêt + suppression si déjà existant
	exec.Command("docker", "stop", req.ContainerName).Run()
	exec.Command("docker", "rm", req.ContainerName).Run()

	network := req.Network
	if network == "" {
		network = "doric"
	}

	args := []string{"run", "-d", "--name", req.ContainerName, "--restart", "unless-stopped", "--network", network}

	for k, v := range req.EnvVars {
		args = append(args, "-e", k+"="+v)
	}
	for host, cont := range req.Ports {
		args = append(args, "-p", host+":"+cont)
	}
	for host, cont := range req.Volumes {
		args = append(args, "-v", host+":"+cont)
	}
	args = append(args, req.Image)

	out, err := exec.Command("docker", args...).CombinedOutput()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": strings.TrimSpace(string(out)) + ": " + err.Error(),
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"container_name": req.ContainerName,
		"id":             strings.TrimSpace(string(out)),
	})
}

// handleContainers — liste tous les containers Doric
func handleContainers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	filter := r.URL.Query().Get("filter") // ex: "doric-db-"
	args := []string{"ps", "-a", "--format", "{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.ID}}"}
	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		json.NewEncoder(w).Encode([]ContainerInfo{})
		return
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		name := parts[0]
		if filter != "" && !strings.HasPrefix(name, filter) {
			continue
		}
		containers = append(containers, ContainerInfo{
			Name:   name,
			Image:  parts[1],
			Status: parts[2],
			ID:     parts[3],
		})
	}
	json.NewEncoder(w).Encode(containers)
}

// ── Main ──────────────────────────────────────────────────────────────────

func main() {
	port := os.Getenv("AGENT_PORT")
	if port == "" {
		port = "50052"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health",          handleHealth)
	mux.HandleFunc("/deploy",          handleDeploy)
	mux.HandleFunc("/run",             handleRun)
	mux.HandleFunc("/control",         handleControl)
	mux.HandleFunc("/logs",            handleLogs)
	mux.HandleFunc("/containers",      handleContainers)
	mux.HandleFunc("/stats",           handleStats)
	mux.HandleFunc("/storage/buckets", handleStorageBuckets)
	mux.HandleFunc("/storage/objects", handleStorageObjects)
	mux.HandleFunc("/storage/upload",  handleStorageUpload)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		// WriteTimeout intentionnellement absent : les streams de deploy peuvent durer longtemps
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("--- Doric Agent v2 démarré sur :%s ---", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}
