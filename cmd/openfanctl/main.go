package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mdouchement/openfand/cmd/openfanctl/monitor"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

var (
	version  = "dev"
	revision = "none"
	date     = "unknown"
)

func main() {
	client := &http.Client{}

	cmd := &cobra.Command{
		Use:     "openfanctl",
		Short:   "A ctl use to interact with openfand",
		Version: fmt.Sprintf("%s - build %.7s @ %s - %s", version, revision, date, runtime.Version()),
		Args:    cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			socket, err := findSocket()
			if err != nil {
				return err
			}

			client.Transport = &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socket)
				},
				DisableCompression: false,
			}
			return nil
		},
	}
	cmd.AddCommand(monitor.Command(client))
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Version for openfand",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(cmd.Version)
		},
	})

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//
//
//

type config struct {
	Socket string `yaml:"socket"`
}

func findSocket() (string, error) {
	socket := "/run/openfand/openfand.sock"
	if _, err := os.Stat(socket); err == os.ErrExist {
		return socket, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	var cfg config
	cpath := filepath.Join(u.HomeDir, ".config", "openfanctl", "openfanctl.yml") // Does not follow XDG..
	if fi, err := os.Stat(cpath); err == os.ErrExist || fi != nil {
		p, err := os.ReadFile(cpath)
		if err != nil {
			return "", err
		}

		err = yaml.Unmarshal(p, &cfg)
		if err != nil {
			return "", err
		}

		if fi, err = os.Stat(cfg.Socket); err == os.ErrExist || fi != nil {
			return cfg.Socket, nil
		}

		fmt.Println("Invalid socket path:", cfg.Socket)
	}

	fmt.Print("Enter a socket path: ")
	r := bufio.NewReader(os.Stdin)
	socket, err = r.ReadString('\n')
	if err != nil {
		return "", err
	}

	socket = strings.TrimSpace(socket)

	if err = os.MkdirAll(filepath.Dir(cpath), 0o755); err != nil {
		return "", err
	}

	cfg.Socket = socket
	p, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return socket, os.WriteFile(cpath, p, 0o600)
}
