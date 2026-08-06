package test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"moria/internal/database"
)

func TestServeCommand(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "moria")
	build := exec.Command("go", "build", "-o", bin, "moria")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	freePort := func(t *testing.T) string {
		t.Helper()
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		_, port, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		return port
	}

	serveOnce := func(t *testing.T, dir string, env []string, port string, args ...string) {
		t.Helper()
		cmd := exec.Command(bin, append([]string{"serve"}, args...)...)
		cmd.Dir = dir
		cmd.Env = env
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}

		healthy := false
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
			resp, err := http.Get("http://127.0.0.1:" + port + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					healthy = true
					break
				}
			}
			time.Sleep(25 * time.Millisecond)
		}
		if !healthy {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("server never became healthy on :%s\n%s", port, out.String())
		}

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("exit after SIGTERM: %v\n%s", err, out.String())
		}
	}

	t.Run("env binding with service links inert", func(t *testing.T) {
		dir := t.TempDir()
		port := freePort(t)
		serveOnce(t, dir, []string{
			"MORIA_LISTEN_PORT=" + port,
			"MORIA_DATABASE_URL=" + newTestDatabaseURL(t),
			"MORIA_PORT=tcp://10.43.0.1:80",
			"MORIA_PORT_80_TCP=tcp://10.43.0.1:80",
			"MORIA_SERVICE_HOST=10.43.0.1",
			"MORIA_SERVICE_PORT=80",
		}, port)
	})

	t.Run("flag beats env", func(t *testing.T) {
		dir := t.TempDir()
		flagPort, envPort := freePort(t), freePort(t)
		serveOnce(t, dir, []string{
			"MORIA_LISTEN_PORT=" + envPort,
			"MORIA_DATABASE_URL=" + newTestDatabaseURL(t),
		}, flagPort, "--listen-port="+flagPort)
	})

	t.Run("config file discovered in cwd", func(t *testing.T) {
		dir := t.TempDir()
		port := freePort(t)
		cfg := fmt.Sprintf(`{"listen-port": %q, "database-url": %q}`,
			port, newTestDatabaseURL(t))
		if err := os.WriteFile(filepath.Join(dir, ".moria.json"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		serveOnce(t, dir, []string{}, port)
	})

	t.Run("env beats config file", func(t *testing.T) {
		dir := t.TempDir()
		jsonPort, envPort := freePort(t), freePort(t)
		cfg := fmt.Sprintf(`{"listen-port": %q, "database-url": %q}`,
			jsonPort, newTestDatabaseURL(t))
		if err := os.WriteFile(filepath.Join(dir, ".moria.json"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		serveOnce(t, dir, []string{"MORIA_LISTEN_PORT=" + envPort}, envPort)
	})

	t.Run("malformed config ignored", func(t *testing.T) {
		dir := t.TempDir()
		port := freePort(t)
		if err := os.WriteFile(filepath.Join(dir, ".moria.json"), []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		serveOnce(t, dir, []string{
			"MORIA_LISTEN_PORT=" + port,
			"MORIA_DATABASE_URL=" + newTestDatabaseURL(t),
		}, port)
	})
}

func TestCreateUserCommand(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "moria")
	build := exec.Command("go", "build", "-o", bin, "moria")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	t.Run("creates the first admin", func(t *testing.T) {
		dbURL := newTestDatabaseURL(t)
		cmd := exec.Command(bin, "create-user",
			"--database-url", dbURL,
			"--username", "cmd-admin",
			"--email", "cmd-admin@example.com",
			"--role", "admin")
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"MORIA_PASSWORD=cmd-password"}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("create-user: %v\n%s", err, out)
		}
		lines := strings.Fields(string(out))
		if len(lines) == 0 {
			t.Fatal("create-user printed no user id")
		}
		id := lines[len(lines)-1]

		cdb, err := database.New(context.Background(), dbURL)
		if err != nil {
			t.Fatal(err)
		}
		defer cdb.Close()
		var role string
		if err := cdb.Get(&role, `SELECT role FROM users WHERE user_id = $1`, id); err != nil {
			t.Fatalf("user %q not found: %v", id, err)
		}
		if role != "admin" {
			t.Errorf("role = %q, want %q", role, "admin")
		}
	})

	t.Run("invalid role fails before touching the database", func(t *testing.T) {
		cmd := exec.Command(bin, "create-user",
			"--database-url", "postgres://moria:x@127.0.0.1:1/moria",
			"--username", "cmd-bad",
			"--email", "cmd-bad@example.com",
			"--role", "superuser")
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"MORIA_PASSWORD=cmd-password"}
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("exit = 0, want failure\n%s", out)
		}
		if !strings.Contains(string(out), `invalid role "superuser"`) {
			t.Errorf("output = %q, want it to name the invalid role", out)
		}
	})
}
