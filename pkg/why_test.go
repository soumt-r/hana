package pkg

import (
	"strings"
	"testing"
)

func TestWhyShowsHowAPackageGotIn(t *testing.T) {
	src := &fakeSource{source: map[string]map[string]string{
		pkgA: {"1.0.0": manifest(pkgA, map[string]string{pkgB: "1.0.0"})},
		pkgB: {"1.0.0": manifest(pkgB, map[string]string{pkgC: "1.0.0"})},
		pkgC: {"1.0.0": manifest(pkgC, nil)},
	}}
	proj := newProject(t)
	in := &Installer{Src: src, Proj: proj}
	if _, err := in.Add(pkgA, nil); err != nil {
		t.Fatal(err)
	}

	chains, err := in.Why(pkgC)
	if err != nil {
		t.Fatal(err)
	}
	if len(chains) != 1 || strings.Join(chains[0], ">") != pkgA+">"+pkgB+">"+pkgC {
		t.Fatalf("chains = %v", chains)
	}

	// b is also added directly: it is reached both ways.
	if _, err := in.Add(pkgB, nil); err != nil {
		t.Fatal(err)
	}
	chains, err = in.Why(pkgB)
	if err != nil {
		t.Fatal(err)
	}
	if len(chains) != 2 {
		t.Fatalf("chains = %v, want a direct one and one through a", chains)
	}

	if _, err := in.Why("github.com/o/nothing"); err == nil {
		t.Error("a package the project does not have was explained")
	}
}

func TestOutdatedListsPackagesWithANewerVersion(t *testing.T) {
	src := &fakeSource{source: map[string]map[string]string{
		pkgA: {"1.0.0": manifest(pkgA, nil)},
		pkgB: {"1.0.0": manifest(pkgB, nil)},
	}}
	proj := newProject(t)
	in := &Installer{Src: src, Proj: proj}
	if _, err := in.Add(pkgA, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := in.Add(pkgB, nil); err != nil {
		t.Fatal(err)
	}
	src.source[pkgB]["1.2.0"] = manifest(pkgB, nil)

	rows, err := (&Installer{Src: src, Proj: proj}).Outdated()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Path != pkgB || rows[0].Current.String() != "1.0.0" || rows[0].Latest.String() != "1.2.0" || !rows[0].Direct {
		t.Fatalf("rows = %+v", rows)
	}
}
