package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto"
)

func main() {
	fmt.Println("=== Testing Daemon gRPC Connection ===\n")

	// 测试连接
	fmt.Println("Attempting to connect to localhost:9091...")
	conn, err := grpc.Dial("localhost:9091",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	fmt.Println("✓ Connection established")
	fmt.Printf("  Connection state: %v\n\n", conn.GetState())

	client := pb.NewDaemonServiceClient(conn)

	fmt.Println("=== Testing Daemon gRPC Agent Operations ===\n")

	// 测试1: 列出所有Agents
	fmt.Println("1. Listing all agents...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	listResp, err := client.ListAgents(ctx, &pb.ListAgentsRequest{})
	duration := time.Since(start)

	if err != nil {
		log.Fatalf("Failed to list agents (took %v): %v", duration, err)
	}

	fmt.Printf("✓ List agents succeeded (took %v)\n", duration)
	fmt.Printf("Found %d agents:\n", len(listResp.Agents))
	for _, agent := range listResp.Agents {
		fmt.Printf("  - %s (%s): status=%s, pid=%d\n",
			agent.Id, agent.Type, agent.Status, agent.Pid)
	}
	fmt.Println()

	// 测试2: 停止agent-002
	fmt.Println("2. Stopping agent-002...")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel2()

	stopResp, err := client.OperateAgent(ctx2, &pb.AgentOperationRequest{
		AgentId:   "agent-002",
		Operation: "stop",
	})
	if err != nil {
		log.Printf("Failed to stop agent: %v", err)
	} else {
		fmt.Printf("Stop result: success=%v, message=%s\n", stopResp.Success, stopResp.ErrorMessage)
	}
	time.Sleep(2 * time.Second)
	fmt.Println()

	// 测试3: 再次列出Agents查看状态
	fmt.Println("3. Listing agents after stop...")
	ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel3()

	listResp2, err := client.ListAgents(ctx3, &pb.ListAgentsRequest{})
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}

	for _, agent := range listResp2.Agents {
		fmt.Printf("  - %s: status=%s, pid=%d\n",
			agent.Id, agent.Status, agent.Pid)
	}
	fmt.Println()

	// 测试4: 启动agent-002
	fmt.Println("4. Starting agent-002...")
	ctx4, cancel4 := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel4()

	startResp, err := client.OperateAgent(ctx4, &pb.AgentOperationRequest{
		AgentId:   "agent-002",
		Operation: "start",
	})
	if err != nil {
		log.Printf("Failed to start agent: %v", err)
	} else {
		fmt.Printf("✓ Start result: success=%v, message=%s\n", startResp.Success, startResp.ErrorMessage)
	}
	// 等待Agent完全启动
	fmt.Println("  Waiting for agent to start...")
	time.Sleep(3 * time.Second)
	fmt.Println()

	// 测试5: 验证启动后的状态
	fmt.Println("5. Verifying agent-002 after start...")
	ctx5, cancel5 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel5()

	listResp3, err := client.ListAgents(ctx5, &pb.ListAgentsRequest{})
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}

	found := false
	for _, agent := range listResp3.Agents {
		if agent.Id == "agent-002" {
			found = true
			if agent.Status == "running" && agent.Pid > 0 {
				fmt.Printf("  ✓ agent-002: status=%s, pid=%d (running)\n", agent.Status, agent.Pid)
			} else {
				fmt.Printf("  ✗ agent-002: status=%s, pid=%d (not running as expected)\n", agent.Status, agent.Pid)
			}
		}
	}
	if !found {
		fmt.Println("  ✗ agent-002 not found in list")
	}
	fmt.Println()

	// 测试6: 重启agent-001
	fmt.Println("6. Restarting agent-001...")
	ctx6, cancel6 := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel6()

	restartResp, err := client.OperateAgent(ctx6, &pb.AgentOperationRequest{
		AgentId:   "agent-001",
		Operation: "restart",
	})
	if err != nil {
		log.Printf("Failed to restart agent: %v", err)
	} else {
		fmt.Printf("✓ Restart result: success=%v, message=%s\n", restartResp.Success, restartResp.ErrorMessage)
	}
	time.Sleep(3 * time.Second)
	fmt.Println()

	// 测试7: 最终状态
	fmt.Println("7. Final agent status...")
	ctx7, cancel7 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel7()

	listResp4, err := client.ListAgents(ctx7, &pb.ListAgentsRequest{})
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}

	for _, agent := range listResp4.Agents {
		fmt.Printf("  - %s: status=%s, pid=%d\n",
			agent.Id, agent.Status, agent.Pid)
	}

	fmt.Println("\n=== Test Complete ===")
}
