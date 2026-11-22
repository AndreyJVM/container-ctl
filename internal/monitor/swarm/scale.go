package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker"
)

const swarmScaleTimeout = 15 * time.Second

// ScaleSwarmService scales a Docker Swarm service to the requested number of replicas.
func ScaleSwarmService(serviceID string, replicas int) error {
	cleanID := strings.TrimSpace(serviceID)
	if cleanID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}
	if replicas < 0 {
		return fmt.Errorf("replicas cannot be negative")
	}

	sockPath := docker.DockerSocketFinder()
	if sockPath != "" {
		client := docker.SocketClientFactory(sockPath)
		if err := scaleSwarmServiceViaSocket(client, cleanID, replicas); err == nil {
			return nil
		}
	}

	return scaleSwarmServiceViaCLI(cleanID, replicas)
}

func scaleSwarmServiceViaSocket(client *http.Client, serviceID string, replicas int) error {
	if client == nil {
		return fmt.Errorf("http client is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), swarmScaleTimeout)
	defer cancel()

	// 1. Inspect existing service to obtain current Version.Index and Spec
	inspectURL := fmt.Sprintf("http://localhost/services/%s", url.PathEscape(serviceID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, inspectURL, nil)
	if err != nil {
		return err
	}

	resp, doErr := client.Do(req)
	if doErr != nil {
		return doErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service inspect returned HTTP %d", resp.StatusCode)
	}

	var svcInfo struct {
		Version struct {
			Index uint64 `json:"Index"`
		} `json:"Version"`
		Spec map[string]interface{} `json:"Spec"`
	}

	if decErr := json.NewDecoder(resp.Body).Decode(&svcInfo); decErr != nil {
		return decErr
	}

	// 2. Modify Mode.Replicated.Replicas in Spec
	mode, ok := svcInfo.Spec["Mode"].(map[string]interface{})
	if !ok || mode == nil {
		mode = make(map[string]interface{})
		svcInfo.Spec["Mode"] = mode
	}

	repl, ok := mode["Replicated"].(map[string]interface{})
	if !ok || repl == nil {
		repl = make(map[string]interface{})
		mode["Replicated"] = repl
	}
	repl["Replicas"] = replicas

	// 3. Update service with new Spec
	bodyBytes, mErr := json.Marshal(svcInfo.Spec)
	if mErr != nil {
		return mErr
	}

	updateURL := fmt.Sprintf("http://localhost/services/%s/update?version=%d", url.PathEscape(serviceID), svcInfo.Version.Index)
	postReq, postErr := http.NewRequestWithContext(ctx, http.MethodPost, updateURL, bytes.NewReader(bodyBytes))
	if postErr != nil {
		return postErr
	}
	postReq.Header.Set("Content-Type", "application/json")

	updateResp, updateDoErr := client.Do(postReq)
	if updateDoErr != nil {
		return updateDoErr
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK && updateResp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("service scale update returned HTTP %d", updateResp.StatusCode)
	}

	return nil
}

func scaleSwarmServiceViaCLI(serviceID string, replicas int) error {
	ctx, cancel := context.WithTimeout(context.Background(), swarmScaleTimeout)
	defer cancel()

	scaleArg := fmt.Sprintf("%s=%d", serviceID, replicas)
	out, err := docker.DockerCLIRunner(ctx, "service", "scale", scaleArg)
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}
