package swarm

import "testing"

func TestParseSwarmCLIInfo(t *testing.T) {
	activeOutput := "active true 3 1"
	active, isMgr, nodes, mgrs := parseSwarmCLIInfo(activeOutput)
	if !active || !isMgr || nodes != 3 || mgrs != 1 {
		t.Errorf("parseSwarmCLIInfo(%q) = (%v, %v, %d, %d); want (true, true, 3, 1)", activeOutput, active, isMgr, nodes, mgrs)
	}

	inactiveOutput := "inactive false 0 0"
	active, _, _, _ = parseSwarmCLIInfo(inactiveOutput)
	if active {
		t.Errorf("expected active=false for inactive output")
	}

	emptyOutput := ""
	active, _, _, _ = parseSwarmCLIInfo(emptyOutput)
	if active {
		t.Errorf("expected active=false for empty output")
	}
}

func TestParseSwarmCLINodes(t *testing.T) {
	output := "node1234567890\thost-1\tmanager\tready\tactive\tLeader\nworker123456789\thost-2\tworker\tdown\tdrain\t\n\n"
	nodes := parseSwarmCLINodes(output)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].ID != "node12345678" || nodes[0].Hostname != "host-1" || !nodes[0].IsLeader {
		t.Errorf("unexpected leader node: %+v", nodes[0])
	}
	if nodes[1].ID != "worker123456" || nodes[1].Role != "worker" || nodes[1].IsLeader {
		t.Errorf("unexpected worker node: %+v", nodes[1])
	}

	// Empty and incomplete lines
	emptyNodes := parseSwarmCLINodes("   \n\nincomplete\tfield\n")
	if len(emptyNodes) != 0 {
		t.Errorf("expected 0 nodes for malformed input, got %d", len(emptyNodes))
	}
}

func TestParseSwarmCLIServices(t *testing.T) {
	output := "svc1234567890\tapi-service\treplicated\t2/2\tapi:v1@sha256:abc\t*:8080->80/tcp\nsvcGlobal12345\tlogger\tglobal\t1/1 (global)\tfluentd:latest\t\n"
	svcs := parseSwarmCLIServices(output)
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services, got %d", len(svcs))
	}
	if svcs[0].Name != "api-service" || svcs[0].RunningReplicas != 2 || svcs[0].TargetReplicas != 2 || svcs[0].Image != "api:v1" {
		t.Errorf("unexpected replicated service: %+v", svcs[0])
	}
	if svcs[1].Name != "logger" || svcs[1].Mode != "global" || svcs[1].RunningReplicas != 1 || svcs[1].TargetReplicas != 1 {
		t.Errorf("unexpected global service: %+v", svcs[1])
	}

	// Empty and incomplete lines
	emptySvcs := parseSwarmCLIServices("   \n\nincomplete\tfield\n")
	if len(emptySvcs) != 0 {
		t.Errorf("expected 0 services for malformed input, got %d", len(emptySvcs))
	}
}
