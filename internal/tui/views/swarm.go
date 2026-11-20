package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/components"
	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
)

// SwarmContext contains data needed to render the Swarm Cluster tab.
type SwarmContext struct {
	Swarm        monitor.SwarmSummary
	CursorIdx    int
	FormatStatus func(isActive bool) string
	Width        int
}

// RenderSwarm renders the Swarm Cluster tab.
func RenderSwarm(ctx SwarmContext) string {
	var s strings.Builder

	s.WriteString(theme.TitleStyle.Render("DOCKER SWARM CLUSTER") + "\n\n")

	swarmStatus := ctx.FormatStatus(ctx.Swarm.IsActive)
	if !ctx.Swarm.IsActive {
		s.WriteString(fmt.Sprintf("  Status: %s\n\n", swarmStatus))
		s.WriteString("  " + theme.DimStyle.Render("Host is not part of an active Swarm cluster.") + "\n")
		s.WriteString("  " + "To initialize a cluster, run: docker swarm init" + "\n")
		return s.String()
	}

	s.WriteString(fmt.Sprintf("  %s %s    %s %d    %s %d\n\n",
		theme.DimStyle.Render("Status:"), swarmStatus,
		theme.DimStyle.Render("Cluster nodes:"), ctx.Swarm.NodesCount,
		theme.DimStyle.Render("Managers:"), ctx.Swarm.ManagersCount,
	))

	contentWidth := max(ctx.Width-4, 88)

	if len(ctx.Swarm.Nodes) > 0 {
		s.WriteString(theme.BoldStyle.Render("  CLUSTER NODES:") + "\n")
		s.WriteString(fmt.Sprintf("  %-14s %-20s %-12s %-10s %-14s %-8s %-10s\n",
			theme.DimStyle.Render("NODE ID"), theme.DimStyle.Render("HOSTNAME"), theme.DimStyle.Render("ROLE"),
			theme.DimStyle.Render("STATE"), theme.DimStyle.Render("AVAILABILITY"), theme.DimStyle.Render("TASKS"), theme.DimStyle.Render("LEADER"),
		))
		s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
		for _, n := range ctx.Swarm.Nodes {
			leaderMark := ""
			if n.IsLeader {
				leaderMark = theme.YellowStyle.Render("[Leader]")
			}
			avail := n.Availability
			if avail == "" {
				avail = "active"
			}
			s.WriteString(fmt.Sprintf("  %-14s %-20s %-12s %-10s %-14s %-8d %-10s\n",
				n.ID, theme.BoldStyle.Render(components.Truncate(n.Hostname, 18)), n.Role, n.State, avail, n.ActiveTasks, leaderMark,
			))
		}
		s.WriteString("\n")
	}

	if len(ctx.Swarm.Services) > 0 {
		s.WriteString(theme.BoldStyle.Render("  SWARM SERVICES:") + "\n")
		s.WriteString(fmt.Sprintf("  %-14s %-20s %-20s %-10s %-12s %s\n",
			theme.DimStyle.Render("SERVICE ID"), theme.DimStyle.Render("NAME"), theme.DimStyle.Render("IMAGE"),
			theme.DimStyle.Render("MODE"), theme.DimStyle.Render("REPLICAS"), theme.DimStyle.Render("PORTS"),
		))
		s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
		for i, svc := range ctx.Swarm.Services {
			prefix := "  "
			if i == ctx.CursorIdx {
				prefix = theme.GreenStyle.Render("► ")
			}

			repStr := svc.Replicas
			if svc.Mode == "replicated" && svc.TargetReplicas > 0 {
				if svc.RunningReplicas == 0 {
					repStr = theme.RedStyle.Render(svc.Replicas)
				} else if svc.RunningReplicas < svc.TargetReplicas {
					repStr = theme.YellowStyle.Render(svc.Replicas)
				} else {
					repStr = theme.GreenStyle.Render(svc.Replicas)
				}
			}

			row := fmt.Sprintf("%-14s %-20s %-20s %-10s %-12s %s",
				svc.ID, theme.BoldStyle.Render(components.Truncate(svc.Name, 18)), components.Truncate(svc.Image, 18),
				svc.Mode, repStr, svc.Ports,
			)

			if i == ctx.CursorIdx {
				s.WriteString(fmt.Sprintf("%s%s\n", prefix, theme.SelectedRowStyle.Render(row)))
			} else {
				s.WriteString(fmt.Sprintf("%s%s\n", prefix, row))
			}
		}
	}

	return s.String()
}
