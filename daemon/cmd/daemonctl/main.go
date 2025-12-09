package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto"
)

const (
	defaultAddress = "localhost:9091"
	defaultTimeout = 30 * time.Second
)

var (
	address = flag.String("address", defaultAddress, "Daemon gRPC server address")
	timeout = flag.Duration("timeout", defaultTimeout, "Request timeout")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `daemonctl - Daemon Agent管理命令行工具

用法:
  daemonctl [选项] <命令> [参数...]

命令:
  list                   列出所有Agent
  start <agent-id>       启动指定的Agent
  stop <agent-id>        停止指定的Agent
  restart <agent-id>     重启指定的Agent
  status <agent-id>      查看指定Agent的状态

选项:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
示例:
  daemonctl list
  daemonctl start agent-001
  daemonctl stop agent-002
  daemonctl restart agent-001
  daemonctl status agent-001
`)
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	// 连接到Daemon
	conn, err := grpc.Dial(*address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法连接到Daemon (%s): %v\n", *address, err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewDaemonServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 执行命令
	switch command {
	case "list":
		err = listAgents(ctx, client)
	case "start":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "错误: start命令需要agent-id参数\n")
			os.Exit(1)
		}
		err = operateAgent(ctx, client, flag.Arg(1), "start")
	case "stop":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "错误: stop命令需要agent-id参数\n")
			os.Exit(1)
		}
		err = operateAgent(ctx, client, flag.Arg(1), "stop")
	case "restart":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "错误: restart命令需要agent-id参数\n")
			os.Exit(1)
		}
		err = operateAgent(ctx, client, flag.Arg(1), "restart")
	case "status":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "错误: status命令需要agent-id参数\n")
			os.Exit(1)
		}
		err = getAgentStatus(ctx, client, flag.Arg(1))
	default:
		fmt.Fprintf(os.Stderr, "错误: 未知命令 '%s'\n", command)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// listAgents 列出所有Agent
func listAgents(ctx context.Context, client pb.DaemonServiceClient) error {
	start := time.Now()
	resp, err := client.ListAgents(ctx, &pb.ListAgentsRequest{})
	if err != nil {
		return fmt.Errorf("列出Agent失败: %w", err)
	}

	duration := time.Since(start)
	fmt.Printf("✓ 成功列出 %d 个Agent (耗时: %v)\n\n", len(resp.Agents), duration)

	if len(resp.Agents) == 0 {
		fmt.Println("  没有找到Agent")
		return nil
	}

	// 打印表格头
	fmt.Printf("%-15s %-10s %-8s %-8s %-20s\n", "AGENT_ID", "TYPE", "STATUS", "PID", "LAST_HEARTBEAT")
	fmt.Println(strings.Repeat("-", 70))

	// 打印每个Agent的信息
	for _, agent := range resp.Agents {
		pid := int(agent.Pid)
		pidStr := "-"
		if pid > 0 {
			pidStr = fmt.Sprintf("%d", pid)
		}

		lastHeartbeat := "-"
		if agent.LastHeartbeat > 0 {
			lastHeartbeat = time.Unix(agent.LastHeartbeat, 0).Format("2006-01-02 15:04:05")
		}

		fmt.Printf("%-15s %-10s %-8s %-8s %-20s\n",
			agent.Id,
			agent.Type,
			agent.Status,
			pidStr,
			lastHeartbeat)
	}

	return nil
}

// operateAgent 操作Agent(启动/停止/重启)
func operateAgent(ctx context.Context, client pb.DaemonServiceClient, agentID, operation string) error {
	start := time.Now()
	resp, err := client.OperateAgent(ctx, &pb.AgentOperationRequest{
		AgentId:   agentID,
		Operation: operation,
	})
	if err != nil {
		return fmt.Errorf("%s Agent失败: %w", operation, err)
	}

	duration := time.Since(start)

	if resp.Success {
		fmt.Printf("✓ %s Agent '%s' 成功 (耗时: %v)\n", strings.Title(operation), agentID, duration)
		if resp.ErrorMessage != "" {
			fmt.Printf("  消息: %s\n", resp.ErrorMessage)
		}
	} else {
		return fmt.Errorf("%s Agent失败: %s", operation, resp.ErrorMessage)
	}

	// 等待一下，然后显示状态
	time.Sleep(500 * time.Millisecond)
	return getAgentStatus(context.Background(), client, agentID)
}

// getAgentStatus 获取指定Agent的状态
func getAgentStatus(ctx context.Context, client pb.DaemonServiceClient, agentID string) error {
	resp, err := client.ListAgents(ctx, &pb.ListAgentsRequest{})
	if err != nil {
		return fmt.Errorf("获取Agent列表失败: %w", err)
	}

	// 查找指定的Agent
	for _, agent := range resp.Agents {
		if agent.Id == agentID {
			fmt.Printf("\nAgent状态:\n")
			fmt.Printf("  ID:            %s\n", agent.Id)
			fmt.Printf("  类型:          %s\n", agent.Type)
			fmt.Printf("  状态:          %s\n", agent.Status)
			fmt.Printf("  PID:           %d\n", agent.Pid)
			fmt.Printf("  版本:          %s\n", agent.Version)
			fmt.Printf("  重启次数:      %d\n", agent.RestartCount)

			if agent.StartTime > 0 {
				startTime := time.Unix(agent.StartTime, 0)
				fmt.Printf("  启动时间:      %s\n", startTime.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  启动时间:      -\n")
			}

			if agent.LastHeartbeat > 0 {
				lastHeartbeat := time.Unix(agent.LastHeartbeat, 0)
				fmt.Printf("  最后心跳:      %s\n", lastHeartbeat.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  最后心跳:      -\n")
			}

			return nil
		}
	}

	return fmt.Errorf("未找到Agent: %s", agentID)
}
