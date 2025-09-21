package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type processSpec struct {
	name    string
	workdir string
	cmd     string
	args    []string
	env     []string
}

type runningProc struct {
	spec processSpec
	cmd  *exec.Cmd
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get wd: %v", err)
	}

	if os.Getenv("USE_DOCKER") == "1" {
		runDockerCompose(root)
		return
	}

	// Prepare environments (lightweight checks/installs)
	if err := ensurePythonEnv(filepath.Join(root, "ml", "service")); err != nil {
		log.Printf("[warn] python env setup failed: %v", err)
	}
	if err := ensureNodeDeps(filepath.Join(root, "web")); err != nil {
		log.Printf("[warn] node deps setup failed: %v", err)
	}

	// Define services
	services := []processSpec{
		{
			name:    "searchd",
			workdir: root,
			cmd:     "go",
			args:    []string{"run", "./services/searchd"},
		},
		{
			name:    "gateway",
			workdir: root,
			cmd:     "go",
			args:    []string{"run", "./services/gateway"},
			env:     []string{"SEARCHD_URL=http://localhost:8090", "GATEWAY_ADDR=:8080"},
		},
		{
			name:    "ml",
			workdir: filepath.Join(root, "ml", "service"),
			cmd:     filepath.Join(filepath.Join(root, "ml", "service"), ".venv", "bin", "uvicorn"),
			args:    []string{"app:app", "--host", "0.0.0.0", "--port", "8000"},
		},
		{
			name:    "web",
			workdir: filepath.Join(root, "web"),
			cmd:     "npm",
			args:    []string{"run", "dev"},
			env:     []string{"NEXT_PUBLIC_GATEWAY_URL=http://localhost:8080"},
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("starting devserver (ctrl-c to stop)")
	runAll(ctx, services)
}

func runDockerCompose(root string) {
	cmd := exec.Command("docker", "compose", "up")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("docker compose up failed: %v", err)
	}
}

func runAll(ctx context.Context, specs []processSpec) {
	var wg sync.WaitGroup
	procs := make([]*runningProc, 0, len(specs))
	for _, s := range specs {
		p, err := startProc(ctx, s)
		if err != nil {
			log.Printf("[%s] failed to start: %v", s.name, err)
			continue
		}
		procs = append(procs, p)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("stopping services...")
	for _, p := range procs {
		_ = terminate(p)
	}
	wg.Wait()
}

func startProc(ctx context.Context, s processSpec) (*runningProc, error) {
	cmd := exec.CommandContext(ctx, s.cmd, s.args...)
	cmd.Dir = s.workdir
	cmd.Env = append(os.Environ(), s.env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil { return nil, err }
	stderr, err := cmd.StderrPipe()
	if err != nil { return nil, err }

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Stream logs with prefixes
	go pipeWithPrefix(s.name, stdout)
	go pipeWithPrefix(s.name, stderr)

	log.Printf("[%s] started pid=%d", s.name, cmd.Process.Pid)
	return &runningProc{spec: s, cmd: cmd}, nil
}

func pipeWithPrefix(name string, r io.Reader) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		fmt.Printf("[%s] %s\n", name, line)
	}
}

func terminate(p *runningProc) error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	// Try graceful SIGTERM, then SIGKILL
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	ch := make(chan error, 1)
	go func() { ch <- p.cmd.Wait() }()
	select {
	case <-time.After(5 * time.Second):
		_ = p.cmd.Process.Kill()
		return errors.New("killed after timeout")
	case err := <-ch:
		return err
	}
}

func ensurePythonEnv(dir string) error {
	venv := filepath.Join(dir, ".venv")
	uvicornBin := filepath.Join(venv, "bin", "uvicorn")
	if _, err := os.Stat(uvicornBin); err == nil {
		return nil
	}
	// Create venv and install deps
	if err := runCmd(dir, nil, "python3", "-m", "venv", ".venv"); err != nil {
		return err
	}
	if err := runCmd(dir, nil, filepath.Join(venv, "bin", "pip"), "install", "-r", "requirements.txt"); err != nil {
		return err
	}
	return nil
}

func ensureNodeDeps(dir string) error {
	nm := filepath.Join(dir, "node_modules")
	if fi, err := os.Stat(nm); err == nil && fi.IsDir() {
		return nil
	}
	return runCmd(dir, nil, "npm", "i")
}

func runCmd(dir string, env []string, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Dir = dir
	if len(env) > 0 {
		c.Env = append(os.Environ(), env...)
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}


