package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/serve"
	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
)

type serveOptions struct {
	addr      string
	workspace string
	reposFile string
	open      bool
	days      int
}

var serveOpts serveOptions

// serveCmd shows the HTML Bartleby has already built for a list of repositories
// behind one landing page. It renders nothing itself: a rebuild is the existing
// `bartleby html`, started in a child process, and only when asked.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve built NERD/NEP documentation for several repositories from one local page",
	Long: `Serve the HTML that "bartleby html" built for a list of repositories, behind one
landing page that lists each repository's NERDs and NEPs with their status.

The repositories are named, one directory per line, in a plain-text file:
$HMD_HOME/.config/bartleby-serve.repos by default, or --repos-file. The
workspace that holds them is $HMD_REPO_HOME by default, or --workspace.

The page only reads what is on disk. A repository whose sources are newer than
its HTML is marked stale, and its Rebuild button runs "bartleby html" in that
repository. The server listens on the loopback interface only.

"bartleby serve suggest" prints repositories you have committed to recently and
have not listed yet.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runServe,
}

var serveSuggestCmd = &cobra.Command{
	Use:           "suggest",
	Short:         "Print repositories with your recent commits that are not in the list",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runServeSuggest,
}

func init() {
	f := serveCmd.PersistentFlags()
	f.StringVar(&serveOpts.addr, "addr", "127.0.0.1:8099", "Address to listen on; must be a loopback address")
	f.StringVar(&serveOpts.workspace, "workspace", "", "Directory holding the repositories (default $HMD_REPO_HOME, else the current directory)")
	f.StringVar(&serveOpts.reposFile, "repos-file", "", "File listing the repositories to serve (default $HMD_HOME/.config/bartleby-serve.repos)")
	serveCmd.Flags().BoolVar(&serveOpts.open, "open", false, "Open the page in the default browser")
	serveSuggestCmd.Flags().IntVar(&serveOpts.days, "days", 30, "Look back this many days for your commits")
	serveCmd.AddCommand(serveSuggestCmd)
}

func serveWorkspace() (string, error) {
	if serveOpts.workspace != "" {
		return serveOpts.workspace, nil
	}
	if v := os.Getenv("HMD_REPO_HOME"); v != "" {
		return os.ExpandEnv(v), nil
	}
	return os.Getwd()
}

func serveReposFile() (string, error) {
	if serveOpts.reposFile != "" {
		return serveOpts.reposFile, nil
	}
	if p := serve.DefaultReposFile(); p != "" {
		return p, nil
	}
	return "", errors.New("no repos file: set HMD_HOME or pass --repos-file")
}

func runServe(cmd *cobra.Command, _ []string) error {
	if err := serve.CheckLoopback(serveOpts.addr); err != nil {
		return err
	}
	ws, err := serveWorkspace()
	if err != nil {
		return err
	}
	file, err := serveReposFile()
	if err != nil {
		return err
	}
	repos, err := serve.LoadRepos(file)
	if err != nil {
		return err
	}
	stderr := cmd.ErrOrStderr()
	if len(repos) == 0 {
		fmt.Fprintf(stderr, "note: %s lists no repositories. Add one directory name per line; `bartleby serve suggest` shows candidates.\n", file)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding this executable to run rebuilds: %w", err)
	}
	srv := &serve.Server{
		Workspace: ws,
		Repos:     repos,
		List: func(workspace string, names []string) (reqtrace.Listing, error) {
			return reqtrace.ListWorkspace(workspace, reqtrace.ListOptions{Workspace: workspace, Repos: names})
		},
		Rebuild: serve.ExecRebuild(exe),
	}

	ln, err := net.Listen("tcp", serveOpts.addr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String() + "/"
	fmt.Fprintf(cmd.OutOrStdout(), "Serving %d repositories from %s at %s (Ctrl-C to stop)\n", len(repos), ws, url)
	if serveOpts.open {
		openBrowser(url, stderr)
	}

	httpSrv := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	errc := make(chan error, 1)
	go func() { errc <- httpSrv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdown)
	}
}

func runServeSuggest(cmd *cobra.Command, _ []string) error {
	ws, err := serveWorkspace()
	if err != nil {
		return err
	}
	file, err := serveReposFile()
	if err != nil {
		return err
	}
	listed, err := serve.LoadRepos(file)
	if err != nil {
		return err
	}
	email, err := serve.GitEmail()
	if err != nil {
		return err
	}
	names, err := serve.Suggest(ws, listed, email, serveOpts.days)
	if err != nil {
		return err
	}
	for _, n := range names {
		fmt.Fprintln(cmd.OutOrStdout(), n)
	}
	if len(names) == 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "No unlisted repositories with commits by you in the last %d days.\n", serveOpts.days)
	}
	return nil
}

// openBrowser is best effort: a failure is a warning, the server still runs.
func openBrowser(url string, stderr interface{ Write([]byte) (int, error) }) {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	if err := exec.Command(opener, url).Start(); err != nil {
		fmt.Fprintf(stderr, "warning: could not open a browser (%v); open %s yourself\n", err, url)
	}
}
