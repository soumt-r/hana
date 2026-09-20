package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/soumt-r/hana/pack"
	"github.com/soumt-r/hana/pkg"

	"github.com/spf13/cobra"
)

var packCmd = &cobra.Command{
	Use:  "pack",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			target = pkg.Platform()
		}
		targetOS, _, ok := strings.Cut(target, "-")
		if !ok {
			fatalf("%s", T("pack.badTarget", target))
		}
		outPath, _ := cmd.Flags().GetString("output")
		if outPath == "" {
			outPath = strings.TrimSuffix(filename, filepath.Ext(filename)) + executableExt(targetOS)
		}
		runtimePath, _ := cmd.Flags().GetString("runtime")
		if runtimePath == "" {
			runtimePath = defaultRuntime(target, targetOS)
		}

		program, lang, libraries := compileFile(filename)
		var encoded bytes.Buffer
		if err := program.Encode(&encoded, lang); err != nil {
			fatalf("%s", T("pack.saveFail", err))
		}
		content := pack.Content{Program: encoded.Bytes(), Modules: libraries}
		var libs []pack.Library
		for _, module := range libraries {
			lib, err := libraryFor(module, target, targetOS)
			if err != nil {
				fatalf("%v", err)
			}
			libs = append(libs, lib)
		}
		embed, _ := cmd.Flags().GetBool("embed")
		if embed {
			content.Libraries = libs
		}

		if err := pack.Write(runtimePath, outPath, content); err != nil {
			fatalf("%s", T("pack.writeFail", err))
		}
		fmt.Printf("✅ %s (%s)\n", outPath, target)
		if !embed {
			written, err := pack.Install(filepath.Dir(outPath), libs)
			if err != nil {
				fatalf("%s", T("pack.exportFail", err))
			}
			for _, path := range written {
				fmt.Printf("   %s\n", path)
			}
		}
	},
}

// libraryFor reads a package's native library for the target platform: the file
// its manifest declares for it, or the legacy <모듈>.<확장자> in the package folder.
func libraryFor(module, target, targetOS string) (pack.Library, error) {
	dir := pkg.Dir(module)
	manifest, err := pkg.Load(dir)
	if err != nil {
		return pack.Library{}, fmt.Errorf("%s", T("pack.badManifest", module, err))
	}
	ext := pkg.LibraryExt(targetOS)
	source := module + ext
	if manifest.HasNative() {
		file, ok := manifest.NativeFor(target)
		if !ok {
			return pack.Library{}, fmt.Errorf("%s", T("pack.noNative", module, target))
		}
		source = filepath.FromSlash(file.File)
	}
	data, err := os.ReadFile(filepath.Join(dir, source))
	if err != nil {
		return pack.Library{}, fmt.Errorf("%s", T("pack.readFail", module, err))
	}
	return pack.Library{Module: module, File: strings.ReplaceAll(module, "/", "_") + ext, Data: data}, nil
}

// defaultRuntime finds the runtime executable next to hana: hana-runtime for
// the platform hana runs on, hana-runtime-<플랫폼> for another target.
func defaultRuntime(target, targetOS string) string {
	name := "hana-runtime"
	if target != pkg.Platform() {
		name += "-" + target
	}
	name += executableExt(targetOS)
	if exe, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	fatalf("%s", T("pack.noRuntime", name))
	return ""
}

func executableExt(goos string) string {
	if goos == "windows" {
		return ".exe"
	}
	return ""
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func init() {
	packCmd.Flags().StringP("output", "o", "", "")
	packCmd.Flags().String("runtime", "", "")
	packCmd.Flags().Bool("embed", false, "")
	packCmd.Flags().String("target", "", "")
	rootCmd.AddCommand(packCmd)
}
