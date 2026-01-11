package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mdouchement/openfand"
	"github.com/spf13/cobra"
)

func Command(client *http.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "monitor",
		Short: "Start the TUI monitor display",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			resp, err := client.Get("http://unix/monitor")
			if err != nil {
				return err
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 { // Should never happen
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
				return fmt.Errorf("sse bad status: %s body=%q", resp.Status, string(b))
			}
			defer resp.Body.Close()

			m := newTUI()
			tui := tea.NewProgram(m, tea.WithAltScreen())

			go func() {
				var event []byte

				for {
					event, err = openfand.ReadSSE(resp.Body)
					if err != nil {
						tui.Quit()
						fmt.Println("ERR:", err)
						os.Exit(1)
					}
					if len(event) == 0 {
						continue
					}

					var evals []openfand.Evaluation
					err = json.Unmarshal(event, &evals)
					if err != nil {
						tui.Quit()
						fmt.Println("ERR:", err)
						os.Exit(1)
					}

					tui.Send(evals)
				}
			}()

			_, err = tui.Run()
			return err
		},
	}
}
