package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewOpenWaitsForDecisionAndEmitsStructuredResult(t *testing.T) {
	result, stderr, opened := runReviewWithBrowser(t, nil)

	require.NoError(t, result.Err)
	require.Contains(t, result.Stdout, `"decision": "approved"`)
	require.Contains(t, stderr, "pxp review listening at http://")
	require.NotEmpty(t, opened)
	require.NotContains(t, stderr, "browser open failed")
}

func TestReviewOpenFailureKeepsManualURLFallback(t *testing.T) {
	openErr := errors.New("no browser available")
	result, stderr, opened := runReviewWithBrowser(t, openErr)

	require.NoError(t, result.Err)
	require.Contains(t, result.Stdout, `"decision": "approved"`)
	require.NotEmpty(t, opened)
	require.Contains(t, stderr, "pxp review browser open failed: no browser available")
	require.Contains(t, stderr, "open "+opened+" manually")
}

func TestBrowserCommandUsesDirectOSLauncher(t *testing.T) {
	tests := []struct {
		goos    string
		command string
		args    []string
	}{
		{goos: "darwin", command: "open", args: []string{"http://127.0.0.1:1234"}},
		{goos: "linux", command: "xdg-open", args: []string{"http://127.0.0.1:1234"}},
		{goos: "windows", command: "rundll32.exe", args: []string{"url.dll,FileProtocolHandler", "http://127.0.0.1:1234"}},
	}

	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			command, args, ok := browserCommand(test.goos, test.args[len(test.args)-1])
			require.True(t, ok)
			require.Equal(t, test.command, command)
			require.Equal(t, test.args, args)
		})
	}

	_, _, ok := browserCommand("plan9", "http://127.0.0.1:1234")
	require.False(t, ok)
}

func runReviewWithBrowser(t *testing.T, openErr error) (commandResult, string, string) {
	t.Helper()
	root := t.TempDir()
	reference := root + "/reference.png"
	actual := root + "/actual.png"
	writeTestPNG(t, reference, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	actualImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	actualImage.Set(0, 0, color.RGBA{R: 255, A: 255})
	writeTestPNG(t, actual, actualImage)

	command := NewCommand()
	reviewCommand, _, err := command.Find([]string{"review"})
	require.NoError(t, err)
	var stdout, stderr bytes.Buffer
	reviewCommand.SetOut(&stdout)
	reviewCommand.SetErr(&stderr)
	reviewCommand.SetContext(context.Background())
	reviewCommand.Flags().Bool("json", true, "test structured output")
	require.NoError(t, reviewCommand.Flags().Set("out", root+"/artifacts"))
	require.NoError(t, reviewCommand.Flags().Set("open", "true"))
	require.NoError(t, reviewCommand.Flags().Set("json", "true"))

	opened := make(chan string, 1)
	opener := func(_ context.Context, url string) error {
		opened <- url
		go func() {
			response, postErr := http.Post(
				url+"/api/feedback",
				"application/json",
				strings.NewReader(`{"decision":"approved"}`),
			)
			if postErr == nil {
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
			}
		}()
		return openErr
	}

	err = runReviewCommandWithBrowser(reviewCommand, []string{reference, actual}, opener)
	result := commandResult{Stdout: stdout.String(), Stderr: stderr.String(), Err: err}
	select {
	case url := <-opened:
		return result, result.Stderr, url
	default:
		t.Fatalf("browser opener was not called: %s", fmt.Sprint(result.Err))
		return result, result.Stderr, ""
	}
}
