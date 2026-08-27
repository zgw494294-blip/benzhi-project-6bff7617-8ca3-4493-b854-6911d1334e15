package main

import (
	"bytes"
	"encoding/json"
	"fieldlingua/internal/application"
	"fieldlingua/internal/domain"
	"fieldlingua/internal/persistence"
	"fieldlingua/internal/transport"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func address() string {
	v := flag.Lookup("addr")
	if v != nil && v.Value.String() != "" {
		return v.Value.String()
	}
	if p := os.Getenv("PORT"); p != "" {
		return "127.0.0.1:" + p
	}
	return "127.0.0.1:19081"
}

func validateAddress(addr string) error {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("监听地址格式错误: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("监听地址必须使用回环 IP")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("监听端口必须为 1 至 65535")
	}
	return nil
}

func selfcheck(addr string) error {
	dataDir, err := os.MkdirTemp("", "fieldlingua-selfcheck-")
	if err != nil {
		return fmt.Errorf("创建自检数据目录: %w", err)
	}
	defer os.RemoveAll(dataDir)
	st := persistence.New(dataDir)
	app := application.New(st)
	srv := transport.New(app)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	go http.Serve(ln, srv.Handler())
	base := "http://" + addr
	client := &http.Client{Timeout: 2 * time.Second}
	if err = waitForHealth(client, base+"/healthz", time.Second); err != nil {
		return err
	}
	h := func(path string, v any) (map[string]any, error) {
		body, marshalErr := json.Marshal(v)
		if marshalErr != nil {
			return nil, marshalErr
		}
		resp, requestErr := client.Post(base+path, "application/json", bytes.NewReader(body))
		if requestErr != nil {
			return nil, requestErr
		}
		defer resp.Body.Close()
		var out map[string]any
		if decodeErr := json.NewDecoder(resp.Body).Decode(&out); decodeErr != nil {
			return nil, fmt.Errorf("解析 %s 响应: %w", path, decodeErr)
		}
		if resp.StatusCode >= 300 {
			return out, fmt.Errorf("%s", resp.Status)
		}
		return out, nil
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	projectID := "sc-" + suffix
	segmentID := "s-" + suffix
	revisionID := "r-" + suffix
	credentialID := "cred-" + suffix
	if _, err = h("/api/projects", map[string]any{"projectID": projectID, "title": "自检项目", "languageVariant": "方言", "ownerID": "tester", "collectionSites": []string{"村落"}, "ethicsStatus": "approved"}); err != nil {
		return err
	}
	if _, err = h("/api/segments", map[string]any{"projectID": projectID, "segment": map[string]any{"segmentID": segmentID, "recordingDigest": "abc", "speakerID": "sp1", "startMillis": 0, "endMillis": 1000, "consentRef": "c1"}, "expectedVersion": 1}); err != nil {
		return err
	}
	if _, err = h("/api/revisions", map[string]any{"projectID": projectID, "revision": map[string]any{"revisionID": revisionID, "segmentID": segmentID, "transcriberID": "t1", "changeNote": "初稿", "transcriptSegments": []map[string]any{{"startMillis": 0, "endMillis": 1000, "speakerID": "sp1", "text": "你好", "annotation": "词"}}}, "expectedVersion": 2}); err != nil {
		return err
	}
	if _, err = h("/api/reviews", map[string]any{"projectID": projectID, "revisionID": revisionID, "reviewerID": "expert", "decision": "pass", "expectedVersion": 3}); err != nil {
		return err
	}
	if _, err = h("/api/releases", map[string]any{"projectID": projectID, "issuedBy": "admin", "credentialID": credentialID, "expectedVersion": 4}); err != nil {
		return err
	}
	verified, err := h("/api/credentials/verify", map[string]any{"credentialID": credentialID})
	if err != nil {
		return err
	}
	verification, ok := verified["verification"].(map[string]any)
	if !ok || verification["verificationStatus"] != "valid" {
		return fmt.Errorf("签发凭据自检未通过")
	}
	fmt.Println("selfcheck passed")
	return nil
}

func waitForHealth(client *http.Client, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			closeErr := resp.Body.Close()
			if resp.StatusCode == http.StatusOK && closeErr == nil {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("健康检查未在限定时间内就绪")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func main() {
	addr := flag.String("addr", "", "监听地址")
	self := flag.Bool("selfcheck", false, "执行端到端自检")
	flag.Parse()
	_ = addr
	a := address()
	if err := validateAddress(a); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *self {
		if e := selfcheck(a); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		return
	}
	st := persistence.New(".fieldlingua-data")
	srv := transport.New(application.New(st))
	fmt.Println("服务监听", a)
	if e := http.ListenAndServe(a, srv.Handler()); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	_ = domain.StatusDraft
}
