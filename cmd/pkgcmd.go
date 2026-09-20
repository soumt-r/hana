package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/soumt-r/hana/pkg"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:  "add",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, version := args[0], (*pkg.Version)(nil)
		if at := strings.LastIndex(path, "@"); at >= 0 {
			v, err := pkg.ParseVersion(path[at+1:])
			if err != nil {
				fatalf("%s", T("add.badVersion", path[at+1:]))
			}
			path, version = path[:at], &v
		}
		in := newInstaller(true)
		if allow, _ := cmd.Flags().GetBool("allow-scripts"); allow {
			in.Approve = map[string]bool{path: true}
		}
		v, err := in.Add(path, version)
		if err != nil {
			fatalPkg(err)
		}
		fmt.Printf("✅ %s\n", T("add.done", path, v))
		lock, _ := pkg.LoadLock(in.Proj.Dir)
		if extra := len(lock) - 1; extra > 0 {
			fmt.Println("   " + T("add.alsoNeeded", extra))
		}
	},
}

var installCmd = &cobra.Command{
	Use:  "install",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		in := newInstaller(false)
		lock, err := in.Install()
		if err != nil {
			fatalPkg(err)
		}
		fmt.Printf("✅ %s\n", T("install.done", len(lock)))
	},
}

var removeCmd = &cobra.Command{
	Use:  "remove",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		in := newInstaller(false)
		removed, err := in.Remove(args[0])
		if err != nil {
			fatalPkg(err)
		}
		if !removed {
			fatalf("%s", T("remove.notThere", args[0]))
		}
		fmt.Printf("✅ %s\n", T("remove.done", args[0]))
	},
}

var listCmd = &cobra.Command{
	Use:  "list",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		in := newInstaller(false)
		lock, err := pkg.LoadLock(in.Proj.Dir)
		if err != nil {
			fatalf("%v", err)
		}
		for _, path := range lock.Paths() {
			mark := " "
			if _, direct := in.Proj.Dependencies[path]; direct {
				mark = "*"
			}
			fmt.Printf("%s %s %s (%s)\n", mark, path, lock[path].Version, shortCommit(lock[path].Commit))
		}
		for path, dir := range in.Proj.Replace {
			fmt.Printf("* %s %s\n", path, T("list.replaced", dir))
		}
		if len(lock) == 0 && len(in.Proj.Replace) == 0 {
			fmt.Println(T("list.empty"))
		}
	},
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// newInstaller opens the project the command runs in. With create, a folder that
// has no hana.json yet gets a project of its own.
func newInstaller(create bool) *pkg.Installer {
	proj, err := pkg.FindProject(".")
	if err != nil {
		fatalf("%v", err)
	}
	if proj == nil {
		if !create {
			fatalf("%s", T("pkg.noProject"))
		}
		dir, err := os.Getwd()
		if err != nil {
			fatalf("%v", err)
		}
		proj = &pkg.Project{Dir: dir}
	}
	return &pkg.Installer{Src: pkg.Git{}, Proj: proj, Log: func(event string, args ...interface{}) {
		switch event {
		case "download":
			fmt.Println("   " + T("pkg.download", args...))
		case "native":
			fmt.Println("   " + T("pkg.native", args...))
		case "script":
			fmt.Println("   " + T("pkg.script", args...))
		case "skipScript":
			fmt.Println("   " + T("pkg.skipScript", args...))
		}
	}, Fetch: fetchHTTPS, Out: os.Stdout}
}

// fetchHTTPS opens an https address for the installer (native libraries).
func fetchHTTPS(url string) (io.ReadCloser, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("%s", T("pkg.httpsOnly"))
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return http.MaxBytesReader(nil, resp.Body, 256<<20), nil
}

// fatalPkg ends the command with a package-manager error in the UI language.
func fatalPkg(err error) {
	var e *pkg.Error
	if errors.As(err, &e) {
		fatalf("%s", T("pkg.err."+string(e.Code), e.Args...))
	}
	fatalf("%v", err)
}

func init() {
	addCmd.Flags().Bool("allow-scripts", false, "")
	for _, c := range []*cobra.Command{addCmd, installCmd, removeCmd, listCmd} {
		rootCmd.AddCommand(c)
	}
}
